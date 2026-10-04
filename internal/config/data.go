package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

/*
 定义一些数据库存放的key值, 以及程序运行时的相关参数配置
*/

// -------------------------System Config-----------------------------------
const (

	// MAXGoroutine max goroutine, 执行spider中对协程的数量限制
	MAXGoroutine = 10

	// FilmSaveCacheThreshold 采集数据保存时直接保存 or 缓存到 redis ( h < 168, 缓存到redis)
	FilmSaveCacheThreshold = 168

	// FilmScanSize 同步影片数据时每次扫描的数据量
	FilmScanSize = 50

	FilmPictureUploadDir = "./storage/upload/gallery"
	// FilmPictureUrlPath 上传/同步图片的访问路径前缀: 既是静态路由, 也是保存到库中的图片链接前缀
	FilmPictureUrlPath = "/upload/pic/poster/"
)

// -------------------------redis key-----------------------------------
const (
	// CategoryTreeKey 分类树 key
	CategoryTreeKey = "CategoryTree"

	// MovieListInfoKey movies分类列表 key
	MovieListInfoKey = "MovieList:Cid%d"

	// MovieDetailKey movie detail影视详情信息 可以
	MovieDetailKey = "MovieDetail:Master"
	// MovieBasicInfoKey 影片基本信息, 简略版本
	MovieBasicInfoKey = "MovieBasicInfo:Cid%d:Id%d"

	// MultipleSiteDetailKey 多站点影片信息存储key
	MultipleSiteDetailKey = "MovieDetail:Slave:%s"

	// SearchInfoTemp redis暂存检索数据信息
	SearchInfoTemp = "Search:SearchInfoTemp"

	// SearchTitle 影片分类标题key
	SearchTitle = "Search:Pid%d:Title"
	// SearchTag 影片剧情标签key
	SearchTag = "Search:Pid%d:%s"

	// VirtualPictureKey 待同步图片临时存储 key
	VirtualPictureKey = "Temp:VirtualPicture"
	// MaxScanCount redis Scan 操作每次扫描的数据量, 每次最多扫描300条数据
	MaxScanCount = 300
)

const (
	AuthUserClaims = "UserClaims"
)

// -------------------------manage 管理后台相关key----------------------------------
const (
	// FilmSourceListKey 旧版采集 API 信息列表key (已迁移至 MySQL collect_sources, 仅供 seed 一次性导入)
	FilmSourceListKey = "Config:Collect:FilmSource"
	// PlayFromSourceKey 旧版的 播放组代码 → 采集接口ID 对应 (已由 players 表取代, 仅供 seed 一次性导入)
	PlayFromSourceKey = "Collect:PlayFrom"
	// ManageConfigExpired 管理配置key 长期有效, 暂定10年
	ManageConfigExpired = time.Hour * 24 * 365 * 10
	// SiteConfigBasic 网站参数配置
	SiteConfigBasic = "SystemConfig:SiteConfig:Basic"

	// ThemeDomainMapKey 域名到主题的映射缓存 key
	ThemeDomainMapKey = "Theme:DomainMap"

	// FilmCrontabKey 定时任务列表信息
	FilmCrontabKey = "Cron:Task:Film"
	// DefaultUpdateSpec 每20分钟执行一次
	DefaultUpdateSpec = "0 */20 * * * ?"

	// DefaultUpdateTime 每次采集最近 3 小时内更新的影片
	DefaultUpdateTime = 3
)

// -------------------------Web API相关redis key-----------------------------------
const (
	// IndexCacheKey , 首页数据缓存
	IndexCacheKey = "IndexCache"
)

// -------------------------Database Connection Params-----------------------------------
const (
	UserIdInitialVal = 10000
)

/*
运行环境相关配置, 默认值适用于本地开发, 可通过环境变量覆盖 (docker compose 部署时使用):
LISTENER_PORT  web服务监听的端口
MYSQL_DSN      mysql连接信息
REDIS_ADDR     redis host:port
REDIS_PASSWORD redis访问密码
REDIS_DB       redis使用第几号库
STARTUP_DELAY  启动时连接数据库前的等待秒数
MIGRATE_ON_START  启动时是否自动执行数据库迁移 (默认 true; false 时需先执行 go run ./cmd/migrate migrate)
以上均可写在 storage/config.env (安装向导生成), 环境变量优先
*/
var (
	ListenerPort  string
	MysqlDsn      string
	RedisAddr     string
	RedisPassword string
	RedisDBNo     int
	StartupDelay  time.Duration
	// MigrateOnStart 启动时自动执行迁移; 设为 false 时只检查, 有未执行的迁移就拒绝启动
	MigrateOnStart bool
)

/*
SEO 发布中心 (internal/publish), 均可通过环境变量覆盖:
SITEMAP_CHUNK_SIZE      每个 sitemap 分片最多的 URL 数
SITEMAP_AUTO_INTERVAL   内容变动后自动重建 sitemap 的检查间隔 (分钟), 0 为不自动重建
RSS_LIMIT               /rss.xml 的影片数
RSS_CACHE_TTL           RSS 缓存秒数
INDEXNOW_ENABLED        是否向 IndexNow 推送变动的 URL
INDEXNOW_KEY            IndexNow 密钥 (同时以 /<key>.txt 提供验证文件)
INDEXNOW_HOST           默认分类方案在「未配置的域名」上的推送域名, 如 www.example.com (其他域名取自网域设置)
INDEXNOW_SCHEME         推送 URL 的协议
INDEXNOW_ENDPOINT       IndexNow 接口
INDEXNOW_BATCH_SIZE     每次推送的 URL 数 (IndexNow 上限 10000)
INDEXNOW_FLUSH_INTERVAL 待推送 URL 的提交间隔 (秒)
*/
var (
	SitemapChunkSize      int
	SitemapAutoInterval   time.Duration
	SitemapDir            string
	RSSLimit              int
	RSSCacheTTL           time.Duration
	IndexNowEnabled       bool
	IndexNowKey           string
	IndexNowHost          string
	IndexNowScheme        string
	IndexNowEndpoint      string
	IndexNowBatchSize     int
	IndexNowFlushInterval time.Duration
)

// loadVars 从环境变量读出全部配置 (由 Load 调用)
func loadVars() {
	ListenerPort = getEnv("LISTENER_PORT", "3601")
	MysqlDsn = getEnv("MYSQL_DSN", "root:123456@(127.0.0.1:3306)/FilmSite?charset=utf8mb4&parseTime=True&loc=Local")
	RedisAddr = getEnv("REDIS_ADDR", "127.0.0.1:6379")
	RedisPassword = getEnv("REDIS_PASSWORD", "")
	RedisDBNo = getEnvInt("REDIS_DB", 0)
	StartupDelay = time.Duration(getEnvInt("STARTUP_DELAY", 20)) * time.Second
	MigrateOnStart = getEnv("MIGRATE_ON_START", "true") != "false"

	SitemapChunkSize = getEnvInt("SITEMAP_CHUNK_SIZE", 50000)
	SitemapAutoInterval = time.Duration(getEnvInt("SITEMAP_AUTO_INTERVAL", 60)) * time.Minute
	SitemapDir = getEnv("SITEMAP_DIR", "./storage/sitemap")
	RSSLimit = getEnvInt("RSS_LIMIT", 50)
	RSSCacheTTL = time.Duration(getEnvInt("RSS_CACHE_TTL", 300)) * time.Second
	IndexNowEnabled = getEnv("INDEXNOW_ENABLED", "false") == "true"
	IndexNowKey = getEnv("INDEXNOW_KEY", "")
	IndexNowHost = getEnv("INDEXNOW_HOST", "")
	IndexNowScheme = getEnv("INDEXNOW_SCHEME", "https")
	IndexNowEndpoint = getEnv("INDEXNOW_ENDPOINT", "https://api.indexnow.org/indexnow")
	IndexNowBatchSize = getEnvInt("INDEXNOW_BATCH_SIZE", 10000)
	IndexNowFlushInterval = time.Duration(getEnvInt("INDEXNOW_FLUSH_INTERVAL", 60)) * time.Second
}

// 缓存 key 统一在此生成, 各处不要自行拼接
const (
	// RSSCacheKey RSS 缓存 (RSS:<方案>:<域名>)
	RSSCacheKey = "RSS"
	// SitemapDirtyKey 内容有变动、sitemap 需要重建 (Sitemap:Dirty:<方案>)
	SitemapDirtyKey = "Sitemap:Dirty"
)

// SitemapDirtyKeyOf 分类方案的 sitemap 需要重建标记
func SitemapDirtyKeyOf(schemeId int64) string {
	return fmt.Sprintf("%s:%d", SitemapDirtyKey, schemeId)
}

// RSSSchemeKeyOf 分类方案的全部 RSS 缓存的前缀 (清除时用)
func RSSSchemeKeyOf(schemeId int64) string {
	return fmt.Sprintf("%s:%d", RSSCacheKey, schemeId)
}

// HomeCacheKeyOf 分类方案的首页缓存 key
func HomeCacheKeyOf(schemeId int64) string {
	return fmt.Sprintf("%s:%d", IndexCacheKey, schemeId)
}

// RSSCacheKeyOf 分类方案 + 域名的 RSS 缓存 key (RSS 内是带域名的绝对 URL)
func RSSCacheKeyOf(schemeId int64, host string) string {
	return fmt.Sprintf("%s:%d:%s", RSSCacheKey, schemeId, host)
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
