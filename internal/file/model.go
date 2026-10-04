package file

import (
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// FileInfo 图片信息对象
type FileInfo struct {
	gorm.Model
	Link         string `json:"link"`         // 图片链接
	Uid          int    `json:"uid"`          // 上传人ID
	RelevanceId  int64  `json:"relevanceId"`  // 关联资源ID
	Type         int    `json:"type"`         // 文件类型 (0 影片封面, 1 用户头像)
	Fid          string `json:"fid"`          // 图片唯一标识, 通常为文件名
	OriginalName string `json:"originalName"` // 上传时的原始档名 (采集同步的图片为空)
	FileType     string `json:"fileType"`     // 文件类型, txt, png, jpg
}

// TableName 设置图片存储表的表名
func (f *FileInfo) TableName() string {
	return "files"
}

// StoragePath 获取文件的保存路径
func (f *FileInfo) StoragePath(uploadDir, accessPrefix string) string {
	var storage string
	switch f.FileType {
	case "jpeg", "jpg", "png", "webp", "gif":
		storage = strings.Replace(f.Link, accessPrefix, fmt.Sprint(uploadDir, "/"), 1)
	default:
	}
	return storage
}

// VirtualPicture 采集入站,待同步的图片信息
type VirtualPicture struct {
	Id   int64  `json:"id"`
	Link string `json:"link"`
}

// ImageTypes 可以上传、在图库中显示的图片类型 (扩展名, 小写)
var ImageTypes = []string{"jpeg", "jpg", "png", "webp", "gif"}

// IsImage 文件名的扩展名是否为允许的图片类型 (不分大小写)
func IsImage(name string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	for _, t := range ImageTypes {
		if ext == t {
			return true
		}
	}
	return false
}
