package database

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"sync"
	"time"
)

// RedisTask 异步缓存策略任务类型
// 有2种类型的任务：更新，删除，新建
type RedisTask struct {
	taskType string
	val      Record
}

type RedisTool struct {
	redisClient *redis.Client
	taskQueue   chan RedisTask
	workerSize  int
	bloomMap    sync.Map
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
	Redis.cachePreheating()
}

func (rt *RedisTool) StartWorkers() {
	for i := 0; i < rt.workerSize; i++ {
		go rt.worker()
	}
}

func (rt *RedisTool) worker() {
	for task := range rt.taskQueue {
		switch task.taskType {
		case "update":
			err := task.val.Delete()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("数据库更新失败")
			}
		case "delete":
			err := task.val.Delete()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("删除失败")
			}
		case "add":
			err := task.val.Add()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("添加失败")
			}
		default:
			log.Log.WithFields(logrus.Fields{
				"error": "未知任务类型",
				"task":  task.taskType,
			}).Error("未知任务类型")
		}
	}
}

func (rt *RedisTool) cacheRepo(repo *GbRepos) error {
	repoJson, err := json.Marshal(*repo)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("序列化仓库失败")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Set(ctx, "repo"+repo.RepoName, repoJson, 24*time.Hour)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("添加仓库失败")
		return result.Err()
	}
	return nil
}

// IfRepoExist 判断仓库是否存在，不会返回gorm.ErrRecordNotFound，只会返回true或false
func (rt *RedisTool) IfRepoExist(name string) (bool, error) {
	// 布隆过滤器判断
	if value, ok := rt.bloomMap.Load("repo" + name); !ok || !value.(bool) {
		return false, nil
	}
	// 从缓存获取
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Get(ctx, "repo"+name)
	if result.Err() != nil && !errors.Is(result.Err(), redis.Nil) {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("获取仓库失败")
		return false, result.Err()
	} else if errors.Is(result.Err(), redis.Nil) {
		// 从数据库获取
		repo := GbRepos{}
		err := repo.GetByStr("repo_name", name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("从数据库获取仓库失败")
			return false, err
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		// 缓存
		err = rt.cacheRepo(&repo)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return true, nil
}

func (rt *RedisTool) AddNewRepo(repo *GbRepos) error {
	ifExist, err := rt.IfRepoExist(repo.RepoName)
	if err != nil {
		return err
	}
	if ifExist {
		return errors.New("仓库已存在")
	}
	// 存入缓存
	err = rt.cacheRepo(repo)
	if err != nil {
		return err
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "add",
		val:      repo,
	}
	// 添加到布隆过滤器
	rt.bloomMap.Store("repo"+repo.RepoName, true)
	return nil
}

func (rt *RedisTool) DeleteRepo(repo *GbRepos) error {
	// 删除缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Del(ctx, "repo"+repo.RepoName)
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
	// 从布隆过滤器删除
	rt.bloomMap.Delete("repo" + repo.RepoName)
	return nil
}

func (rt *RedisTool) UpdateRepo(repo *GbRepos) error {
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
	return nil
}

func (rt *RedisTool) GetRepo(name string) (*GbRepos, error) {
	exist, err := rt.IfRepoExist(name)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, gorm.ErrRecordNotFound
	}
	// 从缓存获取
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()
	result3 := rt.redisClient.Get(ctx3, "repo"+name)
	if result3.Err() != nil && !errors.Is(result3.Err(), redis.Nil) {
		log.Log.WithFields(logrus.Fields{
			"error": result3.Err(),
		}).Error("获取仓库失败")
		return nil, result3.Err()
	} else if errors.Is(result3.Err(), redis.Nil) {
		// 从数据库获取
		repo := GbRepos{}
		err = repo.GetByStr("repo_name", name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("从数据库获取仓库失败")
			return nil, err
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		// 缓存
		err = rt.cacheRepo(&repo)
		if err != nil {
			return nil, err
		}
		return &repo, nil
	} else {
		repo := GbRepos{}
		err = json.Unmarshal([]byte(result3.Val()), &repo)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("反序列化仓库失败")
			return nil, err
		}
		return &repo, nil
	}
}

func (rt *RedisTool) GetAllReposNames() ([]string, error) {
	// 从数据库获取
	temp := GbRepos{}
	records, err := temp.GetAll()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("从数据库获取所有仓库失败")
		return nil, err
	}
	// 将Record接口转化为GbRepos
	var repos []GbRepos
	for _, record := range *records {
		repo, ok := record.(*GbRepos)
		if !ok {
			log.Log.WithFields(logrus.Fields{
				"error": "类型转化失败",
			}).Error("类型转化失败")
			return nil, errors.New("类型转化失败")
		}
		repos = append(repos, *repo)
	}
	var names []string
	for _, repo := range repos {
		names = append(names, repo.RepoName)
	}
	return names, nil
}

func (rt *RedisTool) AddNewMessage(message *Message) error {
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "add",
		val:      message,
	}
	return nil
}

func (rt *RedisTool) DeleteMessage(message *Message) error {
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "delete",
		val:      message,
	}
	return nil
}

func (rt *RedisTool) UpdateMessage(message *Message) error {
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "update",
		val:      message,
	}
	return nil
}

func (rt *RedisTool) GetMessages(fromId, toId int) (*[]*Message, error) {
	// 从数据库获取
	temp := Message{}
	// messages是*[]*Message
	messages, err := temp.GetByFromId(fromId)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("从数据库获取所有消息失败")
		return nil, err
	}
	// 从messages中筛选出时间最近的前十条toId的消息，用message的Time字段排序
	// 从messages中筛选出toId的消息
	var result []*Message
	for _, message := range *messages {
		if message.ToId == toId {
			result = append(result, message)
		}
	}
	// 对result按照Time字段排序
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Time.Before(result[j].Time) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return &result, nil
}

func (rt *RedisTool) cacheCity(city *City) error {
	cityJson, err := json.Marshal(*city)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("序列化城市失败")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Set(ctx, "city"+city.City, cityJson, 24*time.Hour)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("添加城市失败")
		return result.Err()
	}
	return nil
}

func (rt *RedisTool) IfCityExist(cityName string) (bool, error) {
	// 布隆过滤器判断
	if value, ok := rt.bloomMap.Load("city" + cityName); !ok || !value.(bool) {
		return false, nil
	}
	// 从缓存获取
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Get(ctx, "city"+cityName)
	if result.Err() != nil && !errors.Is(result.Err(), redis.Nil) {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("获取城市失败")
		return false, result.Err()
	} else if errors.Is(result.Err(), redis.Nil) {
		// 从数据库获取
		city := City{}
		err := city.GetByStr("city", cityName)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("从数据库获取城市失败")
			return false, err
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		// 缓存
		err = rt.cacheCity(&city)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return true, nil
}

func (rt *RedisTool) AddNewCity(city *City) error {
	ifExist, err := rt.IfCityExist(city.City)
	if err != nil {
		return err
	}
	if ifExist {
		return errors.New("城市已存在")
	}
	// 存入缓存
	err = rt.cacheCity(city)
	if err != nil {
		return err
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "add",
		val:      city,
	}
	// 添加到布隆过滤器
	rt.bloomMap.Store("city"+city.City, true)
	return nil
}

func (rt *RedisTool) DeleteCity(city *City) error {
	// 删除缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Del(ctx, "city"+city.City)
	if result.Err() != nil {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("删除城市失败")
		return result.Err()
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "delete",
		val:      city,
	}
	// 从布隆过滤器删除
	rt.bloomMap.Delete("city" + city.City)
	return nil
}

func (rt *RedisTool) UpdateCity(city *City) error {
	// 更新缓存
	err := rt.cacheCity(city)
	if err != nil {
		return err
	}
	// 异步更新数据库
	rt.taskQueue <- RedisTask{
		taskType: "update",
		val:      city,
	}
	return nil
}

func (rt *RedisTool) GetCity(cityName string) (*City, error) {
	exist, err := rt.IfCityExist(cityName)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, gorm.ErrRecordNotFound
	}
	// 从缓存获取
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := rt.redisClient.Get(ctx, "city"+cityName)
	if result.Err() != nil && !errors.Is(result.Err(), redis.Nil) {
		log.Log.WithFields(logrus.Fields{
			"error": result.Err(),
		}).Error("获取城市失败")
		return nil, result.Err()
	} else if errors.Is(result.Err(), redis.Nil) {
		// 从数据库获取
		city := City{}
		err = city.GetByStr("city", cityName)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("从数据库获取城市失败")
			return nil, err
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		// 缓存
		err = rt.cacheCity(&city)
		if err != nil {
			return nil, err
		}
		return &city, nil
	} else {
		city := City{}
		err = json.Unmarshal([]byte(result.Val()), &city)
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("反序列化城市失败")
			return nil, err
		}
		return &city, nil
	}
}

func (rt *RedisTool) GetAllCities() (*[]*City, error) {
	// 从数据库获取
	temp := City{}
	records, err := temp.GetAll()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("从数据库获取所有城市失败")
		return nil, err
	}
	// 将Record接口转化为City
	var cities []*City
	for _, record := range *records {
		city, ok := record.(*City)
		if !ok {
			log.Log.WithFields(logrus.Fields{
				"error": "类型转化失败",
			}).Error("类型转化失败")
			return nil, errors.New("类型转化失败")
		}
		cities = append(cities, city)
	}
	return &cities, nil
}

func (rt *RedisTool) cachePreheating() {
	names, err := rt.GetAllReposNames()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("缓存预热失败")
	}
	for _, name := range names {
		rt.bloomMap.Store("repo"+name, true)
	}
	cities, err := rt.GetAllCities()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("缓存预热失败")
	}
	if cities == nil {
		return
	}
	for _, city := range *cities {
		rt.bloomMap.Store("city"+city.City, true)
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
