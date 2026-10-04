// Command migrate 手动管理数据库 (与 Laravel artisan migrate 相近, 底层为 goose 与 database/seed):
//
//	go run ./cmd/migrate migrate             执行尚未执行的迁移 (与服务启动时相同)
//	go run ./cmd/migrate status              各迁移的执行状态
//	go run ./cmd/migrate rollback            退回最后一个迁移 (执行它的 Down; 基线不能退回)
//	go run ./cmd/migrate seed                写入初始数据 (幂等, 与服务启动时相同)
//	go run ./cmd/migrate fresh [--seed] [--redis] [--force]
//	                                         删除库中全部表与视图 → 执行全部迁移 → (--seed) 写入初始数据;
//	                                         --redis 同时清空所用的 Redis 库 (站点设置、缓存、检索标签);
//	                                         需输入库名确认, --force 跳过 (GIN_MODE=release 时必须加 --force)
//
// 连线设定与服务相同 (环境变量 MYSQL_DSN、REDIS_ADDR、REDIS_DB ... 优先, 其次为安装向导写入的 storage/config.env).
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"gomaccms/database/migrations"
	"gomaccms/internal/bootstrap"
	"gomaccms/internal/db"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm/logger"
)

// command 解析后的指令
type command struct {
	name               string
	seed, redis, force bool
}

// flagsOf 各指令可用的选项
var flagsOf = map[string][]string{"migrate": nil, "status": nil, "rollback": nil, "seed": nil, "fresh": {"--seed", "--redis", "--force"}}

func parseArgs(args []string) (command, error) {
	if len(args) == 0 {
		return command{}, fmt.Errorf("缺少指令")
	}
	c := command{name: args[0]}
	allowed, ok := flagsOf[c.name]
	if !ok {
		return c, fmt.Errorf("未知的指令 %q", c.name)
	}
	for _, a := range args[1:] {
		found := false
		for _, f := range allowed {
			found = found || a == f
		}
		if !found {
			return c, fmt.Errorf("%s 不支持 %q", c.name, a)
		}
		switch a {
		case "--seed":
			c.seed = true
		case "--redis":
			c.redis = true
		case "--force":
			c.force = true
		}
	}
	return c, nil
}

const usage = `用法: go run ./cmd/migrate <指令> [选项]
  migrate                          执行尚未执行的迁移
  status                           各迁移的执行状态
  rollback                         退回最后一个迁移
  seed                             写入初始数据
  fresh [--seed] [--redis] [--force]  删除全部表后重新迁移`

func main() {
	c, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(c); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run(c command) error {
	bootstrap.Wire()
	if err := db.InitRedisConn(); err != nil {
		return err
	}
	if err := db.InitMysql(); err != nil {
		return err
	}
	db.Mdb.Logger = logger.Default.LogMode(logger.Warn) // 不逐条打印 SQL
	ctx := context.Background()
	switch c.name {
	case "migrate":
		return migrations.Up(ctx)
	case "status":
		return printStatus(ctx)
	case "rollback":
		path, err := migrations.Rollback(ctx)
		if err == nil {
			fmt.Println("已退回:", path)
		}
		return err
	case "seed":
		if err := bootstrap.Seed(); err != nil {
			return err
		}
		fmt.Println("初始数据已写入")
		return nil
	case "fresh":
		return fresh(ctx, c)
	}
	return nil
}

func printStatus(ctx context.Context) error {
	list, err := migrations.Status(ctx)
	if err != nil {
		return err
	}
	pending := 0
	for _, s := range list {
		state := "已执行 " + s.AppliedAt.Local().Format("2006-01-02 15:04:05")
		if s.State == goose.StatePending {
			state, pending = "未执行", pending+1
		}
		fmt.Printf("%-40s %s\n", s.Source.Path, state)
	}
	fmt.Printf("共 %d 个迁移, %d 个未执行\n", len(list), pending)
	return nil
}

// fresh 确认后删除全部表并重新迁移
func fresh(ctx context.Context, c command) error {
	name, err := migrations.Database(ctx)
	if err != nil {
		return err
	}
	if !c.force {
		if os.Getenv("GIN_MODE") == "release" {
			return fmt.Errorf("GIN_MODE=release 时 fresh 必须加 --force")
		}
		what := "库 " + name + " 的全部表与数据"
		if c.redis {
			what += fmt.Sprintf(", 以及 Redis 第 %d 号库", db.Rdb.Options().DB)
		}
		fmt.Printf("将删除%s, 无法恢复。\n请输入库名 %s 确认: ", what, name)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), name) { // 库名在 macOS / Windows 上不分大小写
			return fmt.Errorf("库名不符, 已取消")
		}
	}
	if c.redis {
		if err := db.Rdb.FlushDB(ctx).Err(); err != nil {
			return fmt.Errorf("清空 Redis: %w", err)
		}
		fmt.Println("Redis 已清空")
	}
	if err := migrations.Fresh(ctx); err != nil {
		return err
	}
	if c.seed {
		if err := bootstrap.Seed(); err != nil {
			return err
		}
		fmt.Println("初始数据已写入")
	}
	fmt.Println("完成")
	return nil
}
