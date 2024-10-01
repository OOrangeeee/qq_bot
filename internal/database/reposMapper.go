package database

import "GitHubBot/internal/model"

func AddNewRepo(newRepo *model.GbRepos) error {
	return DB.DataBase.Create(newRepo).Error
}

func DeleteUnscopedRepo(repo *model.GbRepos) error {
	return DB.DataBase.Unscoped().Delete(repo).Error
}

func UpdateRepo(repo *model.GbRepos) error {
	return DB.DataBase.Model(&model.GbRepos{}).Where("id = ?", repo.ID).Updates(map[string]interface{}{
		"repo_name": repo.RepoName,
		"repo_url":  repo.Url,
	}).Error
}

func GetAllRepos() ([]*model.GbRepos, error) {
	var repos []*model.GbRepos
	result := DB.DataBase.Find(&repos)
	return repos, result.Error
}

func GetRepoByName(repoName string) ([]*model.GbRepos, error) {
	var repos []*model.GbRepos
	result := DB.DataBase.Find(&repos, "repo_name = ?", repoName)
	return repos, result.Error
}

func GetAllReposName() ([]string, error) {
	var repos []*model.GbRepos
	result := DB.DataBase.Find(&repos)
	if result.Error != nil {
		return nil, result.Error
	}
	var repoNames []string
	for _, repo := range repos {
		repoNames = append(repoNames, repo.RepoName)
	}
	return repoNames, nil
}
