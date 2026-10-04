package bootstrap

import (
	"gomaccms/internal/collect"
	"gomaccms/internal/config"
	"gomaccms/internal/cron"
	"gomaccms/internal/util"
	"log"
)

// SpiderInit 数据采集相关信息初始化
func SpiderInit() {
	FilmSourceInit()
	CollectCrontabInit()
}

// FilmSourceInit  初始化预存站点信息 提供一些预存采集连Api链接
func FilmSourceInit() {
	if collect.Repo.ExistCollectSourceList() {
		return
	}
	l := []collect.FilmSource{
		{Id: util.GenerateSalt(), Name: "HD(LZ)", Uri: `https://cj.lziapi.com/api.php/provide/vod/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(BF)", Uri: `https://bfzyapi.com/api.php/provide/vod/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false, Interval: 2500},
		{Id: util.GenerateSalt(), Name: "HD(FF)", Uri: `http://cj.ffzyapi.com/api.php/provide/vod/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(OK)", Uri: `https://api.okzyw.net/api.php/provide/vod/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(MD)", Uri: `https://www.mdzyapi.com/api.php/provide/vod/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(LY)", Uri: `https://360zy.com/api.php/provide/vod/at/json`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(SN)", Uri: `https://suoniapi.com/api.php/provide/vod/from/snm3u8/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false, Interval: 2000},
		{Id: util.GenerateSalt(), Name: "HD(DB)", Uri: `https://caiji.dbzy.tv/api.php/provide/vod/from/dbm3u8/at/josn/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
		{Id: util.GenerateSalt(), Name: "HD(TT)", Uri: `https://caiji.dyttzyapi.com/api.php/provide/vod/at/json/`, ResultModel: collect.JsonResult, CollectType: collect.CollectVideo, State: false},
	}
	err := collect.Repo.SaveCollectSourceList(l)
	if err != nil {
		log.Println("SaveSourceApiList Error: ", err)
	}
}

// CollectCrontabInit 初始化系统预定义的定时任务
func CollectCrontabInit() {
	if cron.Repo.Exist() {
		for _, task := range cron.Repo.GetAll() {
			// 旧版的「采集重试」任务 (失效记录已并入采集记录) 直接删除
			if task.Model == 2 {
				cron.Repo.Delete(task.Id)
				continue
			}
			cid, err := cron.AddCron(task.Id, task.Spec)
			if err != nil {
				log.Println("自动任务恢复失败: ", err.Error())
				continue
			}
			task.Cid = cid
			cron.Repo.Update(task)
		}
	} else {
		collectTask := cron.FilmCollectTask{Id: util.GenerateSalt(), Time: config.DefaultUpdateTime, Spec: config.DefaultUpdateSpec,
			Model: 0, State: false, Remark: "每20分钟执行一次已启用站点数据的自动更新"}
		cid, err := cron.AddCron(collectTask.Id, collectTask.Spec)
		if err != nil {
			log.Println("影视更新定时任务添加失败: ", err.Error())
			return
		}
		collectTask.Cid = cid
		cron.Repo.Save(collectTask)
	}

	cron.Scheduler.Start()
}
