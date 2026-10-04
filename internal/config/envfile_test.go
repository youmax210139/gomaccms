package config

import (
	"os"
	"path/filepath"
	"testing"
)

// unsetEnv 在测试期间删除这些环境变量, 结束后恢复
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// useTempDir 切换到临时目录, 结束后清掉文件设置的 key 并恢复默认配置
func useTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Cleanup(func() {
		os.Remove(EnvFilePath)
		Load()
	})
	return dir
}

func TestLoadPrecedence(t *testing.T) {
	useTempDir(t)
	unsetEnv(t, "MYSQL_DSN", "LISTENER_PORT", "REDIS_DB")
	t.Setenv("REDIS_ADDR", "env:6379")
	content := "# 注释\n\nMYSQL_DSN=u:p@tcp(h:1)/d?a=b&c=d\nREDIS_ADDR=file:6379\nLISTENER_PORT=9000\n"
	if err := os.MkdirAll("storage", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(EnvFilePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	Load()

	if MysqlDsn != "u:p@tcp(h:1)/d?a=b&c=d" {
		t.Errorf("MysqlDsn = %q", MysqlDsn)
	}
	if RedisAddr != "env:6379" {
		t.Errorf("RedisAddr = %q, env should win over file", RedisAddr)
	}
	if ListenerPort != "9000" {
		t.Errorf("ListenerPort = %q", ListenerPort)
	}
	if RedisDBNo != 0 {
		t.Errorf("RedisDBNo = %d, want default 0", RedisDBNo)
	}
	if !EnvProvided("REDIS_ADDR") {
		t.Error("REDIS_ADDR comes from the process env")
	}
	if EnvProvided("MYSQL_DSN") {
		t.Error("MYSQL_DSN comes from config.env, not the process env")
	}
}

func TestLoadReloadsRewrittenFile(t *testing.T) {
	useTempDir(t)
	unsetEnv(t, "REDIS_ADDR")

	if err := WriteEnvFile(EnvFilePath, map[string]string{"REDIS_ADDR": "a:1"}); err != nil {
		t.Fatal(err)
	}
	Load()
	if RedisAddr != "a:1" {
		t.Fatalf("RedisAddr = %q, want a:1", RedisAddr)
	}

	if err := WriteEnvFile(EnvFilePath, map[string]string{"REDIS_ADDR": "b:2"}); err != nil {
		t.Fatal(err)
	}
	Load()
	if RedisAddr != "b:2" {
		t.Fatalf("RedisAddr = %q, want b:2 after rewrite", RedisAddr)
	}

	if err := WriteEnvFile(EnvFilePath, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	Load()
	if RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("RedisAddr = %q, want default after key removed", RedisAddr)
	}
}

func TestWriteEnvFile(t *testing.T) {
	dir := useTempDir(t)
	path := filepath.Join(dir, "storage", "config.env")
	if err := WriteEnvFile(path, map[string]string{"B": "2", "A": "x=y"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "A=x=y\nB=2\n" {
		t.Errorf("content = %q", got)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %v, want 0600", info.Mode().Perm())
	}
	if got := readEnvFile(path); got["A"] != "x=y" || got["B"] != "2" {
		t.Errorf("readEnvFile = %v", got)
	}
}

func TestReadEnvFileMissing(t *testing.T) {
	if got := readEnvFile(filepath.Join(t.TempDir(), "nope.env")); len(got) != 0 {
		t.Errorf("missing file should give empty map, got %v", got)
	}
}
