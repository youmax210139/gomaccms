package cron

import (
	"encoding/json"
	"errors"
	"gomaccms/internal/config"
	"gomaccms/internal/db"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) Save(t FilmCollectTask) {
	data, _ := json.Marshal(t)
	db.Rdb.HSet(db.Cxt, config.FilmCrontabKey, t.Id, data)
}

func (r *Repository) GetAll() []FilmCollectTask {
	var tl []FilmCollectTask
	tMap := db.Rdb.HGetAll(db.Cxt, config.FilmCrontabKey).Val()
	for _, v := range tMap {
		var t = FilmCollectTask{}
		_ = json.Unmarshal([]byte(v), &t)
		tl = append(tl, t)
	}
	return tl
}

func (r *Repository) FindById(id string) (FilmCollectTask, error) {
	var ft = FilmCollectTask{}
	if !db.Rdb.HExists(db.Cxt, config.FilmCrontabKey, id).Val() {
		return ft, errors.New(" The task does not exist ")
	}
	data := db.Rdb.HGet(db.Cxt, config.FilmCrontabKey, id).Val()
	err := json.Unmarshal([]byte(data), &ft)
	return ft, err
}

func (r *Repository) Update(t FilmCollectTask) {
	r.Save(t)
}

func (r *Repository) Delete(id string) {
	db.Rdb.HDel(db.Cxt, config.FilmCrontabKey, id)
}

func (r *Repository) Exist() bool {
	return db.Rdb.Exists(db.Cxt, config.FilmCrontabKey).Val() == 1
}
