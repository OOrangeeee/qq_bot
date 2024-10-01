package database

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	"context"
	"encoding/json"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"time"
)

// RedisTask 异步缓存策略任务类型
// 有2种类型的任务：更新，删除，新建
type RedisTask struct {
	taskType string
	val      *model.GbRepos
}

type RedisTool struct {
	redisClient *redis.Client
	taskQueue   chan RedisTask
	workerSize  int
	bloomFilter *bloom.BloomFilter
}

var Redis RedisTool

func InitRedis() {
	// 链接redis
	var dsn string
	switch config.Config.Flags["env"] {
	case "local":
		dsn = config.Config.AppConfig.Redis.DevDsn
	case "online":
		dsn = config.Config.AppConfig.Redis.ProDsn
	default:
		log.Log.WithFields(logrus.Fields{
			"error": "环境变量错误",
		}).Panic("环境变量错误")
	}
	maxTries := 50
	for maxTries > 0 {
		maxTries--
		opt, err := redis.ParseURL(dsn)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("redis链接错误，正在重试...")
			time.Sleep(1 * time.Second)
			continue
		}
		Redis.redisClient = redis.NewClient(opt)
	}
	if Redis.redisClient == nil && maxTries == 0 {
		log.Log.WithFields(logrus.Fields{
			"error": "redis链接失败",
		}).Panic("redis链接失败, 重试次数超过最大重试次数")
	}
	log.Log.WithFields(logrus.Fields{
		"redis": "redis链接成功",
	}).Info("redis链接成功")
	// 初始化异步缓存协程池
	Redis.workerSize = config.Config.AppConfig.Redis.NumOfWorker
	Redis.taskQueue = make(chan RedisTask, config.Config.AppConfig.Redis.TaskChannelSize)
	Redis.StartWorkers()
	// 初始化布隆过滤器
	Redis.bloomFilter = bloom.NewWithEstimates(config.Config.AppConfig.Redis.BloomFilterCapacity, config.Config.AppConfig.Redis.BloomFilterFalsePositiveRate)
	Redis.cachePreheating()
}

func (rt *RedisTool) StartWorkers() {
	for i := 0; i < rt.workerSize; i++ {
		go rt.worker()
	}
}

func (rt *RedisTool) worker() {
	for task := range rt.taskQueue {
		repos, err := GetRepoByName(task.val.RepoName)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("查找仓库失败")
		}
		var repoExist bool
		if len(repos) == 0 {
			repoExist = false
		} else {
			repoExist = true
		}
		switch task.taskType {
		case "update":
			if !repoExist {
				continue
			}
			err = UpdateRepo(task.val)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("更新仓库失败")
			}
		case "delete":
			if !repoExist {
				continue
			}
			err = DeleteUnscopedRepo(task.val)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("删除仓库失败")
			}
		case "add":
			if repoExist {
				continue
			}
			err = AddNewRepo(task.val)
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("添加仓库失败")
			}
		default:
			log.Log.WithFields(logrus.Fields{
				"error": "未知任务类型",
				"task":  task.taskType,
			}).Error("未知任务类型")
		}
	}
}

func (rt *RedisTool) cacheRepo(repo *model.GbRepos) error {
	repoJson, err := json.Marshal(*repo)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("序列化仓库失败")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Set(ctx, repo.RepoName, repoJson, 24*time.Hour)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("添加仓库失败")
		return result.Err()
	}
	return nil
}

func (rt *RedisTool) AddNewRepo(repo *model.GbRepos) error {
	// 存入缓存
	err := rt.cacheRepo(repo)
	if err != nil {
		return err
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "add",
		val:      repo,
	}
	// 添加到布隆过滤器
	rt.bloomFilter.AddString(repo.RepoName)
	return nil
}

func (rt *RedisTool) DeleteRepo(repo *model.GbRepos) error {
	// 删除缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Del(ctx, repo.RepoName)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("删除仓库失败")
		return result.Err()
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "delete",
		val:      repo,
	}
	return nil
}

func (rt *RedisTool) UpdateRepo(repo *model.GbRepos) error {
	// 更新缓存
	err := rt.cacheRepo(repo)
	if err != nil {
		return err
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "update",
		val:      repo,
	}
	rt.bloomFilter.AddString(repo.RepoName)
	return nil
}

func (rt *RedisTool) IfRepoExist(name string) (bool, error) {
	// 布隆过滤器判断
	if !rt.bloomFilter.TestString(name) {
		return false, nil
	}
	// 从缓存中获取判断
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Exists(ctx, name)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("判断仓库是否存在缓存失败")
		return false, result.Err()
	}
	if result.Val() <= 0 {
		// 从数据库判断
		repos, err := GetRepoByName(name)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("从数据库获取仓库失败")
			return false, err
		}
		if len(repos) <= 0 {
			return false, nil
		} else {
			// 存入缓存
			user := repos[0]
			err := rt.cacheRepo(user)
			if err != nil {
				return true, err
			}
			return true, nil
		}
	}
	return true, nil
}

func (rt *RedisTool) GetRepo(name string) (*model.GbRepos, error) {
	exist, err := rt.IfRepoExist(name)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	// 从缓存获取
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()
	result3 := rt.redisClient.Get(ctx3, name)
	if result3.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result3.Err(),
		}).Error("获取仓库失败")
		return nil, result3.Err()
	}
	repo := model.GbRepos{}
	err = json.Unmarshal([]byte(result3.Val()), &repo)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("反序列化仓库失败")
		return nil, err
	}
	return &repo, nil
}

func (rt *RedisTool) cachePreheating() {
	names, err := GetAllReposName()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("缓存预热失败")
	}
	for _, openId := range names {
		rt.bloomFilter.AddString(openId)
	}
}

func (rt *RedisTool) Exit() {
	close(rt.taskQueue)
	err := rt.redisClient.Close()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("关闭redis失败")
	}
}
