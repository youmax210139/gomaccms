package film

import (
	"fmt"
	"gomaccms/internal/collect"
	"gomaccms/internal/db"
	"log"
	"slices"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VodOrigin 影片来源 (vod_origin 表): 采集接口 + 采集站影片ID → 本站影片; 同一接口再次采集时直接对应, 单片更新时据此找回来源
type VodOrigin struct {
	SourceId string `gorm:"column:source_id;primaryKey"`
	OriginId int64  `gorm:"column:origin_id;primaryKey;autoIncrement:false"`
	VodId    int64  `gorm:"column:vod_id"`
	PlayFrom string `gorm:"column:play_from"` // 该采集接口提供的播放组代码, 逗号分隔
}

// TableName 影片来源表表名
func (VodOrigin) TableName() string {
	return "vod_origin"
}

// IngestItem 单部影片的入库结果 (采集进度日志)
type IngestItem struct {
	Name   string `json:"name"`
	Action string `json:"action"` // 新增 / 更新 / 跳过
	Reason string `json:"reason"` // 跳过的原因
}

// IngestResult 一批采集数据的入库结果
type IngestResult struct {
	Added   int
	Updated int
	Skipped int
	Items   []IngestItem
	// NewPictures 新增影片的本站ID → 封面地址 (供开启了同步图片的采集接口同步封面)
	NewPictures map[int64]string
}

// ingestMu 串行化「查重 + 新增」, 避免并发采集同一部片时重复新增
var ingestMu sync.Mutex

// IngestVods 采集入库 (苹果 CMS 式): 每部影片先按来源 (vod_origin), 再按入库重复规则 rule 找到本站已有的影片;
// 依采集接口的「数据操作」决定新增或更新, 依「地址过滤」过滤播放组代码与年份.
// 更新时以播放组代码合并播放地址 (同代码替换, 新代码追加), 并补上新的分类; ml 中的 Mid 为采集站的影片ID,
// Types 为按分类绑定得到的本站分类 (没有 Types 的影片未绑定分类, 不入库)
func IngestVods(s *collect.FilmSource, rule []string, ml []MovieDetail) IngestResult {
	res := IngestResult{NewPictures: map[int64]string{}}
	codes := splitList(s.FilterCode)
	years := splitList(s.FilterYear)
	ingestMu.Lock()
	defer ingestMu.Unlock()
	var changed []int64
	defer func() {
		if len(changed) > 0 {
			VodsChanged(changed, vodSchemes(changed))
		}
	}()
	skip := func(name, reason string) {
		res.Skipped++
		res.Items = append(res.Items, IngestItem{Name: name, Action: "跳过", Reason: reason})
	}
	for _, m := range ml {
		if strings.TrimSpace(m.Name) == "" {
			skip(m.Name, "片名为空")
			continue
		}
		if len(m.Types) == 0 {
			skip(m.Name, "分类未绑定")
			continue
		}
		originId := m.Mid
		vodId := findVod(s.Id, m, rule)
		exists := vodId > 0
		if exists && vodLocked(vodId) {
			skip(m.Name, "已锁定")
			continue
		}
		if exists && s.Operation == collect.OperationAdd {
			skip(m.Name, "已存在 (仅新增)")
			continue
		}
		if !exists && s.Operation == collect.OperationUpdate {
			skip(m.Name, "不存在 (仅更新)")
			continue
		}
		if filterApplies(s.FilterMode, exists) {
			if len(years) > 0 && !slices.Contains(years, NewVod(m).VodYear) {
				skip(m.Name, "年份不在过滤年份中")
				continue
			}
			if len(codes) > 0 {
				m.PlayFrom, m.PlayList = keepCodes(m.PlayFrom, m.PlayList, codes)
				if len(m.PlayList) == 0 {
					skip(m.Name, "没有过滤代码中的播放组")
					continue
				}
			}
		}
		var err error
		if exists {
			if err = mergeVod(vodId, m); err == nil {
				res.Updated++
				res.Items = append(res.Items, IngestItem{Name: m.Name, Action: "更新"})
			}
		} else {
			if vodId, err = insertVod(m); err == nil {
				res.Added++
				res.Items = append(res.Items, IngestItem{Name: m.Name, Action: "新增"})
				if m.Picture != "" {
					res.NewPictures[vodId] = m.Picture
				}
			}
		}
		if err != nil {
			log.Println("Ingest Vod Error:", m.Name, err)
			skip(m.Name, "入库失败: "+err.Error())
			continue
		}
		db.Mdb.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"vod_id", "play_from"})}).
			Create(&VodOrigin{SourceId: s.Id, OriginId: originId, VodId: vodId, PlayFrom: clip(strings.Join(m.PlayFrom, ","), 500)})
		m.Mid = vodId
		if err = addVodTypes([]MovieDetail{m}); err != nil {
			log.Println("Ingest VodType Error:", m.Name, err)
		}
		EnsurePlayers(m.PlayFrom, s.Name)
		changed = append(changed, vodId)
	}
	return res
}

// findVod 找到本站已有的同一部影片: 先按来源, 再按入库重复规则; 没有时返回 0
func findVod(sourceId string, m MovieDetail, rule []string) int64 {
	var id int64
	db.Mdb.Model(&VodOrigin{}).Where("source_id = ? AND origin_id = ?", sourceId, m.Mid).Limit(1).Pluck("vod_id", &id)
	if id > 0 {
		return id
	}
	v := NewVod(m)
	// 片名比对忽略空白与标点 (NameKey); 片名全是标点等无法比对时按原片名
	query := db.Mdb.Model(&Vod{}).Where("vod_name_key = ?", v.VodNameKey)
	if v.VodNameKey == "" {
		query = db.Mdb.Model(&Vod{}).Where("vod_name = ?", v.VodName)
	}
	for _, f := range rule {
		switch f {
		case "year":
			if v.VodYear != "" {
				query = query.Where("vod_year = ?", v.VodYear)
			}
		case "type":
			query = query.Where("type_id = ?", v.TypeId)
		case "area":
			query = query.Where("vod_area = ?", v.VodArea)
		case "douban":
			if v.VodDoubanId > 0 {
				query = query.Where("vod_douban_id = ?", v.VodDoubanId)
			}
		}
	}
	query.Order("vod_id").Limit(1).Pluck("vod_id", &id)
	return id
}

// filterApplies 地址过滤 (过滤代码 / 过滤年份) 是否作用于这次新增或更新
func filterApplies(mode collect.FilterMode, exists bool) bool {
	switch mode {
	case collect.FilterAll:
		return true
	case collect.FilterAdd:
		return !exists
	case collect.FilterUpdate:
		return exists
	}
	return false
}

// keepCodes 只保留代码在 codes 中的播放组
func keepCodes(from FromList, pl MoviePlayList, codes []string) (FromList, MoviePlayList) {
	var f FromList
	var l MoviePlayList
	for i, g := range pl {
		if i < len(from) && slices.Contains(codes, from[i]) {
			f, l = append(f, from[i]), append(l, g)
		}
	}
	return f, l
}

func insertVod(m MovieDetail) (int64, error) {
	v := NewVod(m)
	v.VodId = 0
	if err := db.Mdb.Create(&v).Error; err != nil {
		return 0, err
	}
	return v.VodId, nil
}

// mergeVod 更新已有影片: 按播放组代码合并播放地址, 更新更新状态、时间、评分等; 名称、分类等不覆盖, 没有封面时补上
func mergeVod(vodId int64, m MovieDetail) error {
	var old Vod
	if err := db.Mdb.Where("vod_id = ?", vodId).First(&old).Error; err != nil {
		return err
	}
	from, pl := old.Detail().PlayFrom, old.Detail().PlayList
	for i, g := range m.PlayList {
		code := playCode(m.PlayFrom, i)
		if at := slices.Index(from, code); at >= 0 && at < len(pl) {
			pl[at] = g
			continue
		}
		from, pl = append(from, code), append(pl, g)
	}
	v := NewVod(m)
	values := map[string]any{
		"vod_play_from": strings.Join(from, playGroupSep),
		"vod_play_url":  EncodePlayList(pl),
		"vod_remarks":   v.VodRemarks,
		"vod_state":     v.VodState,
		"vod_time":      max(v.VodTime, old.VodTime),
		"vod_hits":      max(v.VodHits, old.VodHits),
	}
	if v.VodDoubanScore > 0 {
		values["vod_score"], values["vod_douban_score"] = v.VodScore, v.VodDoubanScore
	}
	if old.VodPic == "" && v.VodPic != "" {
		values["vod_pic"] = v.VodPic
	}
	if old.VodDoubanId == 0 && v.VodDoubanId > 0 {
		values["vod_douban_id"] = v.VodDoubanId
	}
	return db.Mdb.Model(&Vod{}).Where("vod_id = ?", vodId).Updates(values).Error
}

func playCode(from FromList, i int) string {
	if i < len(from) && strings.TrimSpace(from[i]) != "" {
		return strings.TrimSpace(from[i])
	}
	return "play"
}

func splitList(s string) []string {
	var l []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			l = append(l, v)
		}
	}
	return l
}

// OriginsOf 影片的来源: 采集接口ID → 采集站影片ID 列表 (单片更新时按来源重新采集)
func OriginsOf(vodIds []int64) map[string][]int64 {
	var l []VodOrigin
	db.Mdb.Where("vod_id IN ?", vodIds).Find(&l)
	m := make(map[string][]int64)
	for _, o := range l {
		m[o.SourceId] = append(m[o.SourceId], o.OriginId)
	}
	return m
}

// deleteVodRelations 删除影片的分类与来源记录
func deleteVodRelations(tx *gorm.DB, vodId int64) error {
	if err := tx.Where("vod_id = ?", vodId).Delete(&VodType{}).Error; err != nil {
		return err
	}
	if err := tx.Where("vod_id = ?", vodId).Delete(&VodI18n{}).Error; err != nil {
		return err
	}
	return tx.Where("vod_id = ?", vodId).Delete(&VodOrigin{}).Error
}

// SourceVodCounts 采集接口的视频数: exclusive 只来自该接口的视频, shared 与其他采集接口共用的视频
func SourceVodCounts(sourceId string) (exclusive, shared int64) {
	var total int64
	db.Mdb.Model(&VodOrigin{}).Where("source_id = ?", sourceId).Distinct("vod_id").Count(&total)
	db.Mdb.Model(&VodOrigin{}).Where("source_id = ?", sourceId).
		Where("NOT EXISTS (SELECT 1 FROM vod_origin o2 WHERE o2.vod_id = vod_origin.vod_id AND o2.source_id <> ?)", sourceId).
		Distinct("vod_id").Count(&exclusive)
	return exclusive, total - exclusive
}

// ClearSourceVods 清空采集接口的全部视频: 只来自该接口的视频整部删除; 与其他接口共用的视频只移除该接口提供的
// 播放组 (其他来源也提供的代码保留) 与来源记录. 返回删除与保留 (移除了播放组) 的视频数
func ClearSourceVods(sourceId string) (deleted, stripped int, err error) {
	ingestMu.Lock()
	defer ingestMu.Unlock()
	var origins []VodOrigin
	if err = db.Mdb.Where("source_id = ?", sourceId).Find(&origins).Error; err != nil {
		return 0, 0, err
	}
	// 同一视频可能对应该接口的多个采集站影片ID, 合并其提供的播放组代码
	codes := make(map[int64][]string)
	var vodIds []int64
	for _, o := range origins {
		if _, ok := codes[o.VodId]; !ok {
			vodIds = append(vodIds, o.VodId)
			codes[o.VodId] = nil
		}
		codes[o.VodId] = append(codes[o.VodId], splitList(o.PlayFrom)...)
	}
	// 删除前取得影片所在的分类方案, 供缓存失效与推送使用
	schemes := vodSchemes(vodIds)
	for _, id := range vodIds {
		var others []VodOrigin
		db.Mdb.Where("vod_id = ? AND source_id <> ?", id, sourceId).Find(&others)
		err = db.Mdb.Transaction(func(tx *gorm.DB) error {
			if len(others) == 0 {
				if err := deleteVodRelations(tx, id); err != nil {
					return err
				}
				return tx.Where("vod_id = ?", id).Delete(&Vod{}).Error
			}
			if err := stripPlayGroups(tx, id, codes[id], others); err != nil {
				return err
			}
			return tx.Where("vod_id = ? AND source_id = ?", id, sourceId).Delete(&VodOrigin{}).Error
		})
		if err != nil {
			return deleted, stripped, err
		}
		if len(others) == 0 {
			deleted++
		} else {
			stripped++
		}
	}
	// 关键字搜索缓存中的结果可能已失效
	db.Mdb.Exec(fmt.Sprintf("TRUNCATE TABLE %s", VodSearch{}.TableName()))
	VodsChanged(vodIds, schemes)
	return deleted, stripped, nil
}

// stripPlayGroups 从共用的视频中移除 codes 播放组 (其他来源也提供的代码保留); codes 为空 (旧记录) 时不移除
func stripPlayGroups(tx *gorm.DB, vodId int64, codes []string, others []VodOrigin) error {
	keep := make(map[string]bool)
	for _, o := range others {
		for _, c := range splitList(o.PlayFrom) {
			keep[c] = true
		}
	}
	var v Vod
	if err := tx.Select("vod_id", "vod_play_from", "vod_play_url").Where("vod_id = ?", vodId).First(&v).Error; err != nil {
		return err
	}
	d := v.Detail()
	var from FromList
	var pl MoviePlayList
	for i, g := range d.PlayList {
		code := playCode(d.PlayFrom, i)
		if slices.Contains(codes, code) && !keep[code] {
			continue
		}
		from, pl = append(from, code), append(pl, g)
	}
	return tx.Model(&Vod{}).Where("vod_id = ?", vodId).
		Updates(map[string]any{"vod_play_from": strings.Join(from, playGroupSep), "vod_play_url": EncodePlayList(pl)}).Error
}

// vodLocked 视频是否已锁定 (锁定后采集不再更新)
func vodLocked(vodId int64) bool {
	var lock int
	db.Mdb.Model(&Vod{}).Where("vod_id = ?", vodId).Limit(1).Pluck("vod_lock", &lock)
	return lock == 1
}
