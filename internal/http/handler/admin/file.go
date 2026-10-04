package admin

import (
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/file"
	"gomaccms/internal/film"
	"gomaccms/internal/http/response"
	"gomaccms/internal/http/session"
	"gomaccms/internal/paging"
	"gomaccms/internal/siteconfig"
	"gomaccms/internal/util"
	"gomaccms/internal/view/inertia"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	gonertia "github.com/romsar/gonertia/v3"
)

// SingleUpload 单文件上传, 暂定为图片上传
func SingleUpload(c *gin.Context) {
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("上传失败, 当前用户信息异常", c)
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.Failed(err.Error(), c)
		return
	}
	link, err := saveUpload(c, fh, uc.UserID)
	if err != nil {
		response.Failed(err.Error(), c)
		return
	}
	response.Success(link, "上传成功", c)
}

// MultipleUpload 批量文件上传
func MultipleUpload(c *gin.Context) {
	uc, ok := session.Claims(c)
	if !ok {
		response.Failed("上传失败, 当前用户信息异常", c)
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		response.Failed(err.Error(), c)
		return
	}
	var links []string
	for _, fh := range form.File["files"] {
		link, err := saveUpload(c, fh, uc.UserID)
		if err != nil {
			response.Failed(err.Error(), c)
			return
		}
		links = append(links, link)
	}
	response.Success(links, "上传成功", c)
}

// saveUpload 以随机文件名保存上传的图片并登记到图库, 返回图片访问地址
func saveUpload(c *gin.Context, fh *multipart.FileHeader, uid uint) (string, error) {
	// 只接受图片 (含 gif 动图); 其他类型 (如 .html) 会被当作网页提供, 不允许上传
	if !file.IsImage(fh.Filename) {
		return "", fmt.Errorf("只能上传图片 (%s)", strings.Join(file.ImageTypes, " / "))
	}
	fileName := fmt.Sprintf("%s/%s%s", config.FilmPictureUploadDir, util.RandomString(8), filepath.Ext(fh.Filename))
	if err := c.SaveUploadedFile(fh, fileName); err != nil {
		return "", err
	}
	return file.Svc.SingleFileUpload(fileName, fh.Filename, int(uid)), nil
}

// galleryItem 图库图片及其来源说明 (手动上传的原始档名 / 影片封面对应的影片)
type galleryItem struct {
	ID           uint   `json:"ID"`
	Link         string `json:"link"`
	Fid          string `json:"fid"`
	OriginalName string `json:"originalName"`
	RelevanceId  int64  `json:"relevanceId"`
	FilmName     string `json:"filmName"`
	CreatedAt    string `json:"createdAt"`
}

// galleryPage 按来源筛选查询一页图库图片, 并补充影片封面对应的影片名称
func galleryPage(page *paging.Page, source string) []galleryItem {
	fl := file.Svc.GetPhotoPage(page, source)
	var mids []int64
	for _, f := range fl {
		if f.RelevanceId > 0 {
			mids = append(mids, f.RelevanceId)
		}
	}
	names := film.Svc.GetNamesByMids(mids)
	items := make([]galleryItem, 0, len(fl))
	for _, f := range fl {
		items = append(items, galleryItem{ID: f.ID, Link: f.Link, Fid: f.Fid, OriginalName: f.OriginalName,
			RelevanceId: f.RelevanceId, FilmName: names[f.RelevanceId], CreatedAt: f.CreatedAt.Format(time.DateTime)})
	}
	return items
}

// gallerySource 规范化来源筛选参数
func gallerySource(c *gin.Context) string {
	switch s := c.Query("source"); s {
	case file.SourceUpload, file.SourcePoster:
		return s
	default:
		return file.SourceAll
	}
}

// DelFile 删除文件(Inertia 渲染, 完成后重定向回图片墙)
func DelFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.DefaultQuery("id", ""), 10, 64)
	if err != nil {
		inertia.Redirect(c, "/manage/file/upload")
		return
	}
	if err = file.Svc.RemoveFileById(uint(id)); err != nil {
		page := paging.Page{PageSize: siteconfig.Svc.AdminPageSize(), Current: 1}
		_ = inertia.RenderManage(c, "File/FileUpload", gonertia.Props{
			"list": galleryPage(&page, file.SourceAll), "page": page, "source": file.SourceAll,
			"errors": formError("图片删除失败: " + err.Error()),
		})
		return
	}
	// 回到删除前的页码与来源筛选
	inertia.Back(c)
}

// FileGallery 图片墙管理页(Inertia 渲染)
func FileGallery(c *gin.Context) {
	page := pageFrom(c.Request.URL.Query())
	source := gallerySource(c)
	_ = inertia.RenderManage(c, "File/FileUpload", gonertia.Props{
		"list":   galleryPage(page, source),
		"page":   page,
		"source": source,
	})
}

// FileList 图库图片分页列表(JSON), 供表单中「从图库选择」图片使用
func FileList(c *gin.Context) {
	current, err := strconv.Atoi(c.DefaultQuery("current", "1"))
	if err != nil || current < 1 {
		current = 1
	}
	page := paging.Page{PageSize: 24, Current: current}
	list := galleryPage(&page, gallerySource(c))
	response.Success(gin.H{"list": list, "page": page}, "图库图片获取成功", c)
}

// FileGalleryRedirect 旧的图库管理占位页地址, 已合并到图片墙 (/manage/file/upload)
func FileGalleryRedirect(c *gin.Context) {
	inertia.Redirect(c, "/manage/file/upload")
}
