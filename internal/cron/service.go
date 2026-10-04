package cron

import (
	"errors"
	"fmt"
	"gomaccms/internal/util"
	"time"
)

var Svc *Service

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) AddFilmCrontab(cv FilmCronVo) error {
	if err := ValidSpec(cv.Spec); err != nil {
		return err
	}
	task := FilmCollectTask{Id: util.GenerateSalt(), Ids: cv.Ids, Time: cv.Time, Spec: cv.Spec, Model: cv.Model, State: cv.State, Remark: cv.Remark}
	cid, err := AddCron(task.Id, task.Spec)
	if err != nil {
		return errors.New(fmt.Sprint("定时任务添加失败: ", err.Error()))
	}
	task.Cid = cid
	s.repo.Save(task)
	return nil
}

func (s *Service) GetFilmCrontab() []CronTaskVo {
	cst := time.FixedZone("UTC", 8*3600)
	var l []CronTaskVo
	tl := s.repo.GetAll()
	for _, t := range tl {
		e := GetEntryById(t.Cid)
		taskVo := CronTaskVo{FilmCollectTask: t, PreV: e.Prev.In(cst).Format(time.DateTime), Next: e.Next.In(cst).Format(time.DateTime)}
		l = append(l, taskVo)
	}
	return l
}

func (s *Service) GetFilmCrontabById(id string) (FilmCollectTask, error) {
	return s.repo.FindById(id)
}

func (s *Service) UpdateFilmCron(t FilmCollectTask) {
	s.repo.Update(t)
}

func (s *Service) DelFilmCrontab(id string) error {
	ft, err := s.repo.FindById(id)
	if err != nil {
		return fmt.Errorf("定时任务删除失败: %w", err)
	}
	RemoveCron(ft.Cid)
	s.repo.Delete(id)
	return nil
}
