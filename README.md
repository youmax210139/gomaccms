# GoMacCMS

基于 Go (Gin) 的影视 CMS: 采集公开影视资源站 (MacCMS 接口), 以 Go `html/template` 主题服务端渲染前台, 并提供 Inertia.js + Vue 3 + Element Plus 管理后台。

- 前台主题在 `themes/`, 按域名切换, 支持 MacCMS 风格的模板标签 (`{maccms:vod}`、`{maccms:type}`、`{maccms:ad}` ...)
- 管理后台在 `admin/`
- 多站群: 每个域名选用一个分类方案, 方案各自有分类、海报、广告、SEO 规则与前台语言
- 后端使用 Gin + GORM + go-redis, 以 gocolly 采集、robfig/cron 定时更新

## 目录结构

```text
cmd/server/          程序入口
cmd/migrate/         数据库管理工具 (migrate / status / rollback / seed / fresh)
database/            goose 版本化迁移 (migrations) 与初始数据 (seed)
internal/            后端代码: 各功能包 (film / collect / cron / user / member / theme / seo / ad ...)、
                     采集引擎 (spider)、HTTP 路由与处理器 (http)、模板与 Inertia 渲染 (view)
themes/default/      前台主题 (templates 模板 + public 静态资源 + lang 语言包)
admin/               管理后台前端 (Inertia + Vue 3 + Vite), 构建输出到 public/build
storage/             运行时上传的图片、网站地图文件
.docker/             Dockerfile 与 nginx 配置, 配合根目录 docker-compose.yml 部署
```

## 本地部署

前台与管理后台都由同一个 Go 程序 (`cmd/server`) 提供, 本地只需准备 MySQL 与 Redis。所有命令均在仓库根目录执行。

### 1. 环境要求

| 依赖 | 版本 |
| --- | --- |
| Go | 1.25+ |
| Node.js | 20+ (仅用于构建管理后台 `admin`) |
| MySQL | 8.x |
| Redis | 6.x+ |

### 2. 准备 MySQL / Redis

创建数据库 `FilmSite` (数据表与初始数据会在启动时自动创建):

```sql
CREATE DATABASE FilmSite DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

也可以用 Docker 启动:

```bash
docker run -d --name gomaccms-mysql -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=FilmSite mysql:8
docker run -d --name gomaccms-redis -p 6379:6379 redis:7
```

### 3. 配置

连接配置通过环境变量覆盖, 默认值位于 `internal/config/data.go`:

| 环境变量 | 说明 | 默认值 |
| --- | --- | --- |
| `MYSQL_DSN` | MySQL 连接串 | `root:123456@(127.0.0.1:3306)/FilmSite?...` |
| `REDIS_ADDR` | Redis 地址 | `127.0.0.1:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis 库编号 | `0` |
| `LISTENER_PORT` | 应用监听端口 | `3601` |
| `STARTUP_DELAY` | 连接数据库前等待秒数 | `20` |
| `MIGRATE_ON_START` | 启动时自动执行数据库迁移 (`false` 时需手动执行, 有未执行的迁移会拒绝启动) | `true` |

### 4. 构建管理后台

```bash
cd admin
npm install
npm run build        # 输出到仓库根目录的 public/build/
```

开发时可改用 `npm run dev` (Vite HMR, 端口 5173), 后端会通过 `public/hot` 自动切换到 dev server。

### 5. 启动

```bash
go run ./cmd/server
```

- 启动时先等待 `STARTUP_DELAY` 秒, 再执行数据库迁移与初始数据 (默认分类方案与分类、会员组、admin 账号、语言), 然后开始监听端口。
- 非 `GIN_MODE=release` 时, `themes/` 下的模板每次请求都会重新解析, 修改模板无需重启。
- 可选: 使用 [air](https://github.com/air-verse/air) 实现 Go 代码热重载。

### 6. 访问

- 前台首页: <http://127.0.0.1:3601/index>
- 管理后台: <http://127.0.0.1:3601/login>, 默认账号/密码 `admin` / `admin` (登录后请立即修改)
- 在后台「采集管理」中启用采集接口并绑定分类, 执行采集后前台才会有影片数据。

## 数据库管理

```bash
go run ./cmd/migrate migrate             # 执行尚未执行的迁移 (启动时也会自动执行)
go run ./cmd/migrate status              # 各迁移的执行状态
go run ./cmd/migrate rollback            # 退回最后一个迁移
go run ./cmd/migrate seed                # 写入初始数据 (幂等)
go run ./cmd/migrate fresh --seed        # 删除全部表后重新迁移并写入初始数据 (会要求输入库名确认)
```

`fresh` 可加 `--redis` 同时清空所用的 Redis 库, `--force` 跳过确认 (`GIN_MODE=release` 时必须加)。

## Docker 部署

仓库根目录可直接用 Docker Compose 部署 (nginx + 应用 + MySQL + Redis), 镜像从源码构建 (`.docker/Dockerfile`)。

```bash
git clone <本仓库地址> gomaccms && cd gomaccms
cp .env.example .env        # 修改 MYSQL_ROOT_PASSWORD / REDIS_PASSWORD / APP_PORT
docker compose up -d --build
docker compose ps           # 等待所有服务变为 healthy
docker compose logs -f app  # 查看应用日志
```

- 前台首页: `http://服务器IP/index` (端口由 `.env` 的 `APP_PORT` 决定, 默认 80)
- 管理后台: `http://服务器IP/login`
- 数据保存在 Docker volume: `mysql_data` (数据库)、`redis_data` (缓存)、`app_static` (上传图片, 挂载到 `/app/storage`)
- 只有 nginx 对外暴露端口; HTTPS 可修改 `.docker/nginx.conf` 或在外层加反向代理 / CDN

更新与维护:

```bash
git pull && docker compose up -d --build   # 更新代码并重建镜像
docker compose down                        # 停止服务 (保留数据)
docker compose down -v                     # 停止并删除所有数据 (谨慎!)
```

服务器内存较小时 Redis 可能因后台保存失败而拒绝写入, 可在宿主机执行 `sysctl vm.overcommit_memory=1`, 或在 `.env` 中调小 `REDIS_MAXMEMORY`。

## 许可

MIT License, 见 [LICENSE](LICENSE)。
