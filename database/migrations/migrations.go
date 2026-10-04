// Package migrations holds the versioned MySQL schema migrations (goose SQL
// files, embedded into the binary). The server applies them at startup
// (MIGRATE_ON_START, default true); cmd/migrate runs the same functions by hand.
//
// New schema changes go into a new NNNNN_description.sql file here; never edit
// a migration that has already been applied. goose records applied versions
// in the goose_db_version table.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"sort"
	"strings"
	"testing/fstest"

	"gomaccms/internal/db"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// baseline 第一个迁移 (00001_schema.sql, 建立全部表) 的版本; 退回它等于删除全部表, rollback 不做 (请用 fresh)
const baseline = 1

// ngramParser 全文索引的 ngram 分词器子句; MariaDB 没有 ngram (也会执行 /*!50100 */ 注释), 迁移时去掉
var ngramParser = regexp.MustCompile("\\s*/\\*!50100 WITH PARSER `ngram` \\*/")

// sourceFS 迁移文件: MySQL 原样; MariaDB 去掉 ngram parser, 建普通全文索引 (搜索改用 LIKE, 见 film.textMatch)。
// 迁移文件本身不改, 已安装的数据库不受影响。
func sourceFS(mariaDB bool) (fs.FS, error) {
	if !mariaDB {
		return files, nil
	}
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	out := fstest.MapFS{}
	for _, n := range names {
		b, err := fs.ReadFile(files, n)
		if err != nil {
			return nil, err
		}
		out[n] = &fstest.MapFile{Data: ngramParser.ReplaceAll(b, nil)}
	}
	return out, nil
}

// newProvider 以 db.Mdb 的连线建立 goose provider
func newProvider() (*goose.Provider, *sql.DB, error) {
	sqlDB, err := db.Mdb.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("migrations: get sql.DB: %w", err)
	}
	fsys, err := sourceFS(db.IsMariaDB)
	if err != nil {
		return nil, nil, err
	}
	p, err := goose.NewProvider(goose.DialectMySQL, sqlDB, fsys)
	if err != nil {
		return nil, nil, fmt.Errorf("migrations: init goose: %w", err)
	}
	return p, sqlDB, nil
}

// Up applies every pending migration to db.Mdb's MySQL database.
func Up(ctx context.Context) error {
	p, _, err := newProvider()
	if err != nil {
		return err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrations: up: %w", err)
	}
	for _, r := range results {
		log.Printf("migrations: applied %s (%s)", r.Source.Path, r.Duration)
	}
	version, err := p.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrations: get version: %w", err)
	}
	log.Printf("migrations: database schema at version %d", version)
	return nil
}

// Pending 尚未执行的迁移数 (MIGRATE_ON_START=false 时启动前检查)
func Pending(ctx context.Context) (int, error) {
	list, err := Status(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range list {
		if s.State == goose.StatePending {
			n++
		}
	}
	return n, nil
}

// Status 每个迁移的执行状态
func Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	p, _, err := newProvider()
	if err != nil {
		return nil, err
	}
	list, err := p.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: status: %w", err)
	}
	return list, nil
}

// Rollback 退回最后一个已执行的迁移 (执行它的 Down); 基线迁移不能退回
func Rollback(ctx context.Context) (string, error) {
	p, _, err := newProvider()
	if err != nil {
		return "", err
	}
	version, err := p.GetDBVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("migrations: get version: %w", err)
	}
	if version <= baseline {
		return "", fmt.Errorf("migrations: 已在基线版本 %d, 不能再回滚 (要清空重建请用 fresh)", version)
	}
	r, err := p.Down(ctx)
	if err != nil {
		return "", fmt.Errorf("migrations: down: %w", err)
	}
	return r.Source.Path, nil
}

// Database 当前连线的库名
func Database(ctx context.Context) (string, error) {
	var name sql.NullString
	err := db.Mdb.WithContext(ctx).Raw("SELECT DATABASE()").Scan(&name).Error
	if err == nil && !name.Valid {
		err = fmt.Errorf("没有选择数据库")
	}
	return name.String, err
}

// Fresh 删除当前库的全部表与视图 (不执行 Down, 与 Laravel migrate:fresh 相同), 再执行全部迁移
func Fresh(ctx context.Context) error {
	sqlDB, err := db.Mdb.DB()
	if err != nil {
		return fmt.Errorf("migrations: get sql.DB: %w", err)
	}
	if err := dropAll(ctx, sqlDB); err != nil {
		return err
	}
	return Up(ctx)
}

// dropAll 在同一条连线上关闭外键检查后删除全部表与视图
func dropAll(ctx context.Context, sqlDB *sql.DB) error {
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrations: conn: %w", err)
	}
	defer conn.Close()
	rows, err := conn.QueryContext(ctx, "SHOW FULL TABLES")
	if err != nil {
		return fmt.Errorf("migrations: list tables: %w", err)
	}
	var tables, views []string
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			rows.Close()
			return err
		}
		if kind == "VIEW" {
			views = append(views, quoteIdent(name))
		} else {
			tables = append(tables, quoteIdent(name))
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("migrations: list tables: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS = 1")
	if len(views) > 0 {
		if _, err := conn.ExecContext(ctx, "DROP VIEW IF EXISTS "+strings.Join(views, ", ")); err != nil {
			return fmt.Errorf("migrations: drop views: %w", err)
		}
	}
	if len(tables) > 0 {
		if _, err := conn.ExecContext(ctx, "DROP TABLE IF EXISTS "+strings.Join(tables, ", ")); err != nil {
			return fmt.Errorf("migrations: drop tables: %w", err)
		}
	}
	log.Printf("migrations: dropped %d tables, %d views", len(tables), len(views))
	return nil
}

// quoteIdent MySQL 标识符 (表名) 的引用
func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// source 内嵌的一个迁移文件
type source struct {
	Version int64
	Path    string
}

// sources 内嵌的迁移文件, 按版本排序 (不需要连线, 供测试检查)
func sources() ([]source, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	var out []source
	for _, n := range names {
		v, err := goose.NumericComponent(n)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		out = append(out, source{Version: v, Path: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}
