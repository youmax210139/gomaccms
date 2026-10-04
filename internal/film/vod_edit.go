package film

import (
	"crypto/md5"
	"errors"
	"fmt"
	"gomaccms/internal/db"
	"gomaccms/internal/i18n"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// VodPlayGroup 后台编辑的一个播放组: 播放器代码 + 每行一集 (集名$地址, 只有地址时自动命名)
type VodPlayGroup struct {
	From string `json:"from"`
	Text string `json:"text"`
}

// VodEdit 后台视频编辑页的数据: 原文栏位 + 主分类 + 播放组 + 各语言译文
type VodEdit struct {
	Id         int64              `json:"id"`
	Cid        int64              `json:"cid"`
	Name       string             `json:"name"`
	Sub        string             `json:"sub"`
	Letter     string             `json:"letter"`
	Class      string             `json:"class"`
	Pic        string             `json:"pic"`
	Actor      string             `json:"actor"`
	Director   string             `json:"director"`
	Writer     string             `json:"writer"`
	Remarks    string             `json:"remarks"`
	Pubdate    string             `json:"pubdate"`
	Area       string             `json:"area"`
	Lang       string             `json:"lang"`
	Year       string             `json:"year"`
	State      string             `json:"state"`
	Content    string             `json:"content"`
	Status     int                `json:"status"`
	Level      int                `json:"level"`
	Lock       int                `json:"lock"`
	Hits       int64              `json:"hits"`
	Score      float64            `json:"score"`
	PlayGroups []VodPlayGroup     `json:"playGroups"`
	I18n       map[string]VodText `json:"i18n"`
	// Base 打开编辑页时采集会更新的栏位的指纹; 保存时不一致表示期间又采集过, 拒绝保存以免覆盖新数据
	Base string `json:"base"`
}

// collectedHash 采集会更新、编辑页也会写入的栏位 (播放组、备注、状态、人气、评分) 的指纹
func collectedHash(v Vod) string {
	s := strings.Join([]string{v.VodPlayFrom, v.VodPlayUrl, v.VodRemarks, v.VodState,
		strconv.FormatInt(v.VodHits, 10), strconv.FormatFloat(v.VodScore, 'f', -1, 64)}, "\x00")
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// vodEditColumns 后台编辑会写入的 vod 栏位 (不含 vod_time: 编辑不算内容更新)
var vodEditColumns = []string{"type_id", "type_id_1", "type_name", "vod_name", "vod_name_key", "vod_sub", "vod_letter",
	"vod_class", "vod_pic", "vod_actor", "vod_director", "vod_writer", "vod_remarks", "vod_pubdate", "vod_area", "vod_lang",
	"vod_year", "vod_state", "vod_content", "vod_status", "vod_level", "vod_lock", "vod_hits", "vod_score",
	"vod_play_from", "vod_play_url"}

var fullYear = regexp.MustCompile(`^[12][0-9]{3}$`)

// vodEditOf vod 记录转为编辑页数据
func vodEditOf(v Vod) VodEdit {
	e := VodEdit{Id: v.VodId, Cid: v.TypeId, Name: v.VodName, Sub: v.VodSub, Letter: v.VodLetter, Class: v.VodClass,
		Pic: v.VodPic, Actor: v.VodActor, Director: v.VodDirector, Writer: v.VodWriter, Remarks: v.VodRemarks,
		Pubdate: v.VodPubdate, Area: v.VodArea, Lang: v.VodLang, Year: v.VodYear, State: v.VodState,
		Content: v.VodContent, Status: v.VodStatus, Level: v.VodLevel, Lock: v.VodLock, Hits: v.VodHits, Score: v.VodScore}
	e.Base = collectedHash(v)
	var from []string
	if v.VodPlayFrom != "" {
		from = strings.Split(v.VodPlayFrom, playGroupSep)
	}
	if strings.TrimSpace(v.VodPlayUrl) == "" {
		return e
	}
	// 按原样拆开 (不经 ConvertPlayUrl: 它会截断含 $ 的地址、给没有集名的项目加占位名), 保存时才能还原
	for i, g := range strings.Split(v.VodPlayUrl, playGroupSep) {
		// 采集数据末尾多余的 # 与空播放组不显示 (保存时随之去掉)
		var lines []string
		for _, item := range strings.Split(g, "#") {
			if strings.TrimSpace(item) != "" {
				lines = append(lines, item)
			}
		}
		if len(lines) == 0 {
			continue
		}
		// 播放器代码比播放组少时, 与前台一致按 play<N> 命名
		code := fmt.Sprintf("play%d", i+1)
		if i < len(from) && from[i] != "" {
			code = from[i]
		}
		e.PlayGroups = append(e.PlayGroups, VodPlayGroup{From: code, Text: strings.Join(lines, "\n")})
	}
	return e
}

// parsePlayText 一个播放组的文字 (每行一集: 集名$地址, 或只有地址) 转为 集名$地址#… 格式; 空行忽略.
// 以第一个 $ 分开集名与地址 (地址可以含 $), 只有地址的行原样保存 (前台显示为占位集名)
func parsePlayText(text string) (string, error) {
	var items []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		link := line
		if name, rest, ok := strings.Cut(line, "$"); ok {
			link = strings.TrimSpace(rest)
			line = strings.TrimSpace(name) + "$" + link
		}
		if link == "" || strings.Contains(line, "#") {
			return "", fmt.Errorf("播放地址格式错误: %q (每行一集: 集名$地址, 不能包含 #)", line)
		}
		items = append(items, line)
	}
	return strings.Join(items, "#"), nil
}

// applyTo 校验编辑页数据并写入 vod 记录 (分类由 SaveVodEdit 处理)
func (e VodEdit) applyTo(v *Vod) error {
	name := clip(e.Name, 255)
	switch {
	case e.Base != "" && e.Base != collectedHash(*v):
		return errors.New("打开编辑页之后这部视频又被采集更新了 (播放地址、备注等), 请重新载入页面后再修改")
	case name == "":
		return errors.New("片名不能为空")
	case e.Year != "" && !fullYear.MatchString(strings.TrimSpace(e.Year)):
		return errors.New("年份必须是 4 位数字")
	case e.Level < 0 || e.Level > 9:
		return errors.New("推荐等级为 0-9")
	case e.Score < 0 || e.Score > 10:
		return errors.New("评分为 0-10")
	case e.Hits < 0:
		return errors.New("人气不能为负数")
	case e.Status != 0 && e.Status != 1, e.Lock != 0 && e.Lock != 1:
		return errors.New("审核 / 锁定状态无效")
	}
	var from, groups []string
	for _, g := range e.PlayGroups {
		items, err := parsePlayText(g.Text)
		if err != nil {
			return err
		}
		if items == "" {
			continue
		}
		code := strings.TrimSpace(g.From)
		if code == "" || strings.Contains(code, "$") {
			return errors.New("每个播放组都要选择播放器")
		}
		from, groups = append(from, code), append(groups, items)
	}
	v.VodName, v.VodNameKey, v.VodSub, v.VodLetter = name, NameKey(name), clip(e.Sub, 255), clip(e.Letter, 10)
	v.VodClass, v.VodPic, v.VodActor, v.VodDirector = clip(e.Class, 255), clip(e.Pic, 1024), clip(e.Actor, 255), clip(e.Director, 255)
	v.VodWriter, v.VodRemarks, v.VodPubdate, v.VodArea = clip(e.Writer, 255), clip(e.Remarks, 255), clip(e.Pubdate, 100), clip(e.Area, 255)
	v.VodLang, v.VodYear, v.VodState, v.VodContent = clip(e.Lang, 255), strings.TrimSpace(e.Year), clip(e.State, 255), e.Content
	v.VodStatus, v.VodLevel, v.VodLock, v.VodHits, v.VodScore = e.Status, e.Level, e.Lock, e.Hits, e.Score
	v.VodPlayFrom, v.VodPlayUrl = strings.Join(from, playGroupSep), strings.Join(groups, playGroupSep)
	return nil
}

// VodSchemes 影片所在的分类方案 (没有分类时为空)
func VodSchemes(id int64) []int64 {
	return vodSchemes([]int64{id})
}

// LoadVodEdit 后台编辑页的视频数据
func (s *Service) LoadVodEdit(id int64) (VodEdit, error) {
	var v Vod
	if err := db.Mdb.Where("vod_id = ?", id).Limit(1).Find(&v).Error; err != nil || v.VodId == 0 {
		return VodEdit{}, errors.New("视频不存在")
	}
	e := vodEditOf(v)
	e.I18n = VodTranslations(id)
	return e, nil
}

// SaveVodEdit 保存后台编辑: 原文栏位、主分类 (默认方案的视频分类)、播放组与各语言译文 (只处理请求中的已知语言)
func (s *Service) SaveVodEdit(e VodEdit) error {
	var v Vod
	if err := db.Mdb.Where("vod_id = ?", e.Id).Limit(1).Find(&v).Error; err != nil || v.VodId == 0 {
		return errors.New("视频不存在")
	}
	oldCid := v.TypeId
	if err := e.applyTo(&v); err != nil {
		return err
	}
	c, err := CategoryRepo.Find(e.Cid)
	if err != nil || c.Type != CategoryVideo || c.SchemeId != DefaultSchemeId {
		return errors.New("请选择视频分类")
	}
	v.TypeId, v.TypeName, v.TypeId1 = c.Id, clip(c.Name, 60), c.Pid
	if c.Pid == 0 {
		v.TypeId1 = c.Id
	}
	known := map[string]bool{}
	for _, l := range i18n.Svc.List() {
		known[l.Code] = true
	}
	texts := map[string]VodText{}
	for lang, t := range e.I18n {
		if known[lang] {
			texts[lang] = t
		}
	}
	before := vodSchemes([]int64{v.VodId})
	err = db.Mdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Vod{}).Where("vod_id = ?", v.VodId).Select(vodEditColumns).Updates(&v).Error; err != nil {
			return err
		}
		// 主分类改变: 移除原主分类的关联, 新分类由 addVodTypes 补上
		if oldCid != v.TypeId {
			if err := tx.Where("vod_id = ? AND type_id = ?", v.VodId, oldCid).Delete(&VodType{}).Error; err != nil {
				return err
			}
		}
		return saveVodTranslations(tx, v.VodId, texts)
	})
	if err != nil {
		return err
	}
	if err := addVodTypes([]MovieDetail{v.Detail()}); err != nil {
		return err
	}
	// 剧情、地区、年份等可能改了, 为影片的每个分类重新登记检索标签
	var rows []VodType
	db.Mdb.Where("vod_id = ?", v.VodId).Find(&rows)
	for _, r := range rows {
		info := v.Search()
		info.Cid, info.Pid = r.TypeId, r.TypeId1
		SearchRepo.SaveSearchTag(info)
	}
	schemes := before
	for _, id := range vodSchemes([]int64{v.VodId}) {
		if !slices.Contains(schemes, id) {
			schemes = append(schemes, id)
		}
	}
	VodsChanged([]int64{v.VodId}, schemes)
	return nil
}
