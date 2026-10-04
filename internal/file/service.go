package file

import (
	"errors"
	"fmt"
	"gomaccms/internal/config"
	"gomaccms/internal/paging"
	"gomaccms/internal/util"
	"io/fs"
	"path/filepath"
	"strings"
)

var Svc *Service

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SingleFileUpload 记录单张上传图片的信息
func (s *Service) SingleFileUpload(fileName, originalName string, uid int) string {
	var f = FileInfo{Link: fmt.Sprint(config.FilmPictureUrlPath, filepath.Base(fileName)), Uid: uid, Type: 0, OriginalName: filepath.Base(originalName)}
	f.Fid = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	f.FileType = strings.TrimPrefix(filepath.Ext(fileName), ".")
	s.repo.SaveGallery(f)
	return f.Link
}

// 图库来源筛选: 后台手动上传 (无关联影片) / 采集同步的影片封面 (relevance_id 为影片 mid)
const (
	SourceAll    = ""
	SourceUpload = "upload"
	SourcePoster = "poster"
)

// GetPhotoPage 获取系统内的图片分页信息, source 为来源筛选
func (s *Service) GetPhotoPage(page *paging.Page, source string) []FileInfo {
	return s.repo.GetFileInfoPage(ImageTypes, source, page)
}

// RemoveFileById 删除文件信息
func (s *Service) RemoveFileById(id uint) error {
	f := s.repo.GetFileInfoById(id)
	// 磁盘上的文件已不存在 (被手动删除或存储卷被清空) 时仍删除记录, 否则这条记录永远删不掉
	err := util.RemoveFile(f.StoragePath(config.FilmPictureUploadDir, config.FilmPictureUrlPath))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	s.repo.DelFileInfo(id)
	return nil
}

// 注意: SaveVirtualPic/SyncFilmPicture 没有做成 file.Service 方法,
// 唯一的调用方 internal/spider 作为后台采集引擎直接调用 file.Repo,
// 见 internal/spider/spider.go 的 syncFilmPicture()。
