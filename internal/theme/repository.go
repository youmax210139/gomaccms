package theme

import (
	"gomaccms/internal/db"
	"gomaccms/internal/siteconfig"

	"gorm.io/gorm"
)

var Repo *Repository

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) Create(d *Domain) error {
	return db.Mdb.Create(d).Error
}

// Update 修改域名的设定 (文字栏位见 UpdateText)
func (r *Repository) Update(d *Domain) error {
	// Select 显式列出列, 使空字符串也会被写入 (Updates(struct) 默认跳过零值)
	return db.Mdb.Model(&Domain{}).Where("id = ?", d.ID).
		Select("updated_at", "domain", "theme", "state", "scheme_id", "logo", "service_email", "analytics_code").
		Updates(d).Error
}

// UpdateText 只更新站点文字: 网站名称、SEO、法律信息及其各语言译文
func (r *Repository) UpdateText(id uint, si siteconfig.SiteInfo) error {
	res := db.Mdb.Model(&Domain{}).Where("id = ?", id).
		Select("updated_at", "site_name", "seo_title", "keyword", "seo_description", "legal_info", "i18n").
		Updates(&Domain{SiteInfo: si})
	if res.Error == nil && res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return res.Error
}

// UpdateState 只更新网站状态
func (r *Repository) UpdateState(id uint, state bool) error {
	res := db.Mdb.Model(&Domain{}).Where("id = ?", id).Update("state", state)
	if res.Error == nil && res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return res.Error
}

// Delete 物理删除: domain 列是唯一索引, 软删除会让同一域名无法再次添加
func (r *Repository) Delete(id uint) error {
	return db.Mdb.Unscoped().Delete(&Domain{}, id).Error
}

func (r *Repository) GetByDomain(domain string) (*Domain, error) {
	var d Domain
	err := db.Mdb.Where("domain = ?", domain).First(&d).Error
	return &d, err
}

func (r *Repository) GetAll() ([]Domain, error) {
	var dl []Domain
	err := db.Mdb.Order("domain ASC").Find(&dl).Error
	return dl, err
}
