package worker

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config WorkerAgent 配置，从环境变量加载，所有字段均有默认值
type Config struct {
	Addr             string // gRPC 监听地址，默认 :9090
	RedisAddr        string // Redis 地址，默认 localhost:6379
	RedisPass        string // Redis 密码（可选）
	RedisDB          int    // Redis DB 编号，默认 1（Worker 注册与调度专用），从环境变量 WORKER_REDIS_DB 加载
	LogLevel         string // 日志级别 debug/info/warn/error，默认 "info"
	LiveKitHost      string // LiveKit 服务地址，e.g. "http://localhost:7880"
	LiveKitAPIKey    string // LiveKit API Key
	LiveKitAPISecret string // LiveKit API Secret
	EmuRunnerPath    string // EmuRunner 二进制路径，默认 "./emurunner"
}

// MustLoad 从环境变量加载 Worker 配置，缺失时使用默认值
func MustLoad() *Config {
	loadWorkerEnv()

	return &Config{
		Addr:             envOrDefault("WORKER_ADDR", ":9090"),
		RedisAddr:        envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPass:        os.Getenv("REDIS_PASS"),
		RedisDB:          envOrDefaultInt("WORKER_REDIS_DB", 1),
		LogLevel:         envOrDefault("LOG_LEVEL", "info"),
		LiveKitHost:      envOrDefault("LIVEKIT_HOST", "http://localhost:7880"),
		LiveKitAPIKey:    envOrDefault("LIVEKIT_API_KEY", ""),
		LiveKitAPISecret: envOrDefault("LIVEKIT_API_SECRET", ""),
		EmuRunnerPath:    envOrDefault("EMURUNNER_PATH", "./emurunner"),
	}
}

// loadWorkerEnv 从 .env 文件加载环境变量（与 Control Plane 共用同一个 .env 文件）
func loadWorkerEnv() {
	path := findEnvFile()
	if path == "" {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: cannot open .env file %s: %v\n", path, err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 1 {
			fmt.Fprintf(os.Stderr, "warn: %s:%d: invalid line, skipping\n", path, lineNo)
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		if commentIdx := strings.IndexByte(value, '#'); commentIdx >= 0 {
			value = strings.TrimSpace(value[:commentIdx])
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

}

func findEnvFile() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for range 5 {
		path := filepath.Join(wd, ".env")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return ""
}

func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envOrDefaultInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
