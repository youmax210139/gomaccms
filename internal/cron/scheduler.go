package cron

import (
	"errors"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

var (
	Scheduler *cron.Cron = CreateCron()
	// TaskRunner 定时任务触发时实际执行采集的函数, 由 bootstrap 注入为 spider.RunCronTask
	// (spider 依赖 cron 的任务数据, cron 不能反向 import spider)
	TaskRunner func(ft FilmCollectTask)
)

// CreateCron 创建定时任务
func CreateCron() *cron.Cron {
	return cron.New(cron.WithSeconds())
}

// AddCron 添加定时任务
func AddCron(id, spec string) (cron.EntryID, error) {
	// 校验 spec 表达式的有效性
	if err := ValidSpec(spec); err != nil {
		return -99, errors.New(fmt.Sprint("定时任务添加失败,Cron表达式校验失败: ", err.Error()))
	}
	return Scheduler.AddFunc(spec, func() {
		// 通过 Id 获取任务相关数据
		ft, err := Repo.FindById(id)
		if err != nil {
			log.Println("FilmCollectCron Exec Failed: ", err)
		}
		// 开启对系统中已启用站点的自动更新
		if ft.State && TaskRunner != nil {
			TaskRunner(ft)
		}
	})

}

// RemoveCron 删除定时任务
func RemoveCron(id cron.EntryID) {
	// 通过定时任务EntryID移出对应的定时任务
	Scheduler.Remove(id)
}

// GetEntryById 返回定时任务的相关时间信息
func GetEntryById(id cron.EntryID) cron.Entry {
	//log.Printf("CronInfo: %+v\n", Scheduler.Entries())
	//log.Println("Corn Next Execute Time:", Scheduler.Entry(id).Next.Format(time.DateTime))
	return Scheduler.Entry(id)
}

// ValidSpec 校验cron表达式是否有效 不能精确到秒
func ValidSpec(spec string) error {
	// 自定义解释器
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	//if _, err := parser.Parse(spec); err != nil {
	//	return err
	//}
	_, err := parser.Parse(spec)
	return err
}
