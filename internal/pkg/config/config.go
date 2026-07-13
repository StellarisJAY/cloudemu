package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 全局配置，从环境变量加载，所有字段均有默认值
type Config struct {
	Addr              string        // 服务监听地址，默认 :8080
	DSN               string        // PostgreSQL 连接串
	RedisAddr         string        // Redis 地址
	RedisPass         string        // Redis 密码（可选）
	RedisDB           int           // Redis DB 编号，默认 0（业务数据：captcha/room state/limiter）
	WorkerRedisDB     int           // Redis DB 编号（Worker 注册与调度专用），默认 1
	JWTSecret         []byte        // JWT 签名密钥（生产环境务必更换）
	MinioEndpoint     string        // MinIO 服务地址
	MinioAccessKey    string        // MinIO AccessKey
	MinioSecretKey    string        // MinIO SecretKey
	MinioBucket       string        // MinIO 桶名
	MinioUseSSL       bool          // MinIO 是否使用 SSL 连接
	LogDir            string        // 日志文件目录，默认 "logs"
	LogLevel          string        // 日志级别 debug/info/warn/error，默认 "info"
	LogJSON           bool          // 日志输出 JSON 格式，默认 false（Text 格式）
	WorkerGRPCTimeout time.Duration // Worker gRPC 调用超时，默认 5s
	SMTPHost          string        // SMTP 服务器地址，如 smtp.qq.com
	SMTPPort          int           // SMTP 端口号，默认 587
	SMTPUser          string        // SMTP 登录账号
	SMTPPass          string        // SMTP 登录密码或授权码
	SMTPFrom          string        // 发件人显示地址，如 "CloudEmu <noreply@example.com>"
	SMTPUseTLS        bool          // SMTP 是否使用 TLS 连接
	FrontendBaseURL   string        // 前端基础 URL，用于构造密码重置链接，如 http://localhost:5173
}

// MustLoad 从环境变量加载配置，缺失时使用默认值
// 启动时自动加载工作目录及其父目录中的 .env 文件（如存在）
func MustLoad() *Config {
	loadEnv()

	cfg := &Config{
		Addr:              envOrDefault("ADDR", ":8080"),
		DSN:               envOrDefault("DSN", "host=localhost user=postgres password=postgres dbname=cloudemu port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
		RedisAddr:         envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPass:         os.Getenv("REDIS_PASS"),
		RedisDB:           envOrDefaultInt("REDIS_DB", 0),
		WorkerRedisDB:     envOrDefaultInt("WORKER_REDIS_DB", 1),
		JWTSecret:         []byte(envOrDefault("JWT_SECRET", "dev-secret-change-in-production")),
		MinioEndpoint:     envOrDefault("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:    envOrDefault("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:    envOrDefault("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:       envOrDefault("MINIO_BUCKET", "cloudemu"),
		MinioUseSSL:       os.Getenv("MINIO_USE_SSL") == "true",
		LogDir:            envOrDefault("LOG_DIR", "logs"),
		LogLevel:          envOrDefault("LOG_LEVEL", "info"),
		LogJSON:           os.Getenv("LOG_JSON") == "true",
		WorkerGRPCTimeout: envOrDefaultDuration("WORKER_GRPC_TIMEOUT", 5*time.Second),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          envOrDefaultInt("SMTP_PORT", 587),
		SMTPUser:          os.Getenv("SMTP_USER"),
		SMTPPass:          os.Getenv("SMTP_PASS"),
		SMTPFrom:          envOrDefault("SMTP_FROM", ""),
		SMTPUseTLS:        os.Getenv("SMTP_USE_TLS") == "true",
		FrontendBaseURL:   envOrDefault("FRONTEND_BASE_URL", "http://localhost:5173"),
	}
	return cfg
}

// loadEnv 从当前目录向上查找 .env 文件，解析并设置环境变量
// 仅在环境变量尚未设置时（os.Getenv 为空）才从 .env 取值，
// 保证 OS 环境变量优先级高于 .env 文件
func loadEnv() {
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

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 KEY=VALUE
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 1 {
			fmt.Fprintf(os.Stderr, "warn: %s:%d: invalid line, skipping\n", path, lineNo)
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		// 去除行尾的 # 注释（不含引号保护，简单场景足够）
		if commentIdx := strings.IndexByte(value, '#'); commentIdx >= 0 {
			value = strings.TrimSpace(value[:commentIdx])
		}

		// 仅在环境变量未设置时才加载
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// findEnvFile 从当前工作目录向上递归查找 .env 文件，最多向上查找 4 层
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
			break // 到达文件系统根
		}
		wd = parent
	}
	return ""
}

// envOrDefault 获取环境变量，如果为空则返回默认值
func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// envOrDefaultInt 获取整数类型环境变量，如果为空或解析失败则返回默认值
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

// envOrDefaultDuration 获取 Duration 类型环境变量（支持 "5s", "10s", "1m" 等格式），为空或解析失败返回默认值
func envOrDefaultDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func (c *Config) String() string {
	return fmt.Sprintf("Addr=%s DSN=%s Redis=%s Minio=%s", c.Addr, c.DSN, c.RedisAddr, c.MinioEndpoint)
}
