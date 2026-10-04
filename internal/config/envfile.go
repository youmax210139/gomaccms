package config

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnvFilePath 安装向导写入的连接配置 (KEY=VALUE), 优先级低于环境变量、高于代码默认值
const EnvFilePath = "./storage/config.env"

// fileKeys 上一次 Load 从 EnvFilePath 设置到环境变量的 key (重新 Load 时先撤销)
var fileKeys = map[string]bool{}

func init() { Load() }

// Load 把 EnvFilePath 中环境变量里还没有的 key 设置到环境变量, 再读出全部配置。
// 安装向导重写 config.env 后再次调用即生效; 进程启动时就有的环境变量始终优先。
func Load() {
	for k := range fileKeys {
		os.Unsetenv(k)
	}
	fileKeys = map[string]bool{}
	for k, v := range readEnvFile(EnvFilePath) {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
			fileKeys[k] = true
		}
	}
	loadVars()
}

// EnvProvided 该 key 是否由进程环境变量提供 (而不是来自 config.env)
func EnvProvided(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok && !fileKeys[key]
}

// readEnvFile 读取 KEY=VALUE 文件: 忽略空行与 # 注释, 只按第一个 = 切分; 文件不存在时返回空 map
func readEnvFile(path string) map[string]string {
	values := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return values
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) != "" {
			values[strings.TrimSpace(k)] = v
		}
	}
	return values
}

// WriteEnvFile 以 KEY=VALUE (按 key 排序) 覆盖写入 path, 权限 0600
func WriteEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k + "=" + values[k] + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600) // 文件已存在时 WriteFile 不改权限
}
