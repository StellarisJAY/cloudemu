// CloudEmu Control Plane 服务入口
// 负责依赖注入装配、数据库自动迁移、服务启动
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/cache"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/captcha"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	grpcclient "github.com/StellarisJAY/cloudemu/internal/control-plane/grpc"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/handler"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/repo"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/router"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/scheduler"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/service"
	"github.com/StellarisJAY/cloudemu/internal/pkg/config"
	"github.com/StellarisJAY/cloudemu/internal/pkg/email"
	"github.com/StellarisJAY/cloudemu/internal/pkg/logging"

	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.MustLoad()

	// 初始化 slog 日志：stdout + 按天轮转日志文件双向输出
	slog.SetDefault(logging.MustNew(cfg))

	// 判断是否为开发模式（Gorm 日志策略：开发输出所有 SQL，生产仅慢查询和错误）
	devMode := cfg.LogLevel == "debug"

	// 创建 Gorm 适配 slog 的 Logger
	gormLog := logging.NewGormLogger(slog.Default(), devMode)

	// 连接 PostgreSQL，禁用外键约束自动创建（按项目约定不使用 FK）
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   gormLog,
	})
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	// 连接 Redis DB 0（业务数据：captcha/room state/limiter）
	rds := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	// 连接 Redis DB 1（Worker 注册与调度专用）
	workerRds := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.WorkerRedisDB,
	})

	// 开发阶段使用 Gorm AutoMigrate 自动建表/更新字段
	if err := db.AutoMigrate(
		&model.User{},
		&model.EmailVerification{},
		&model.RefreshToken{},
		&model.PasswordReset{},
		&model.Friend{},
		&model.Room{},
		&model.RoomPlayer{},
		&model.Rom{},
	); err != nil {
		slog.Error("failed to migrate", "error", err)
		os.Exit(1)
	}

	// 连接 MinIO（失败不影响启动，ROM 上传功能不可用）
	minioAdapter, err := repo.NewMinioAdapter(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		slog.Warn("minio connection failed, rom upload disabled", "error", err)
	} else {
		// 确保桶存在，不存在则自动创建
		if err := minioAdapter.EnsureBucket(context.Background(), cfg.MinioBucket); err != nil {
			slog.Warn("minio bucket ensure failed, rom upload disabled", "error", err, "bucket", cfg.MinioBucket)
		} else {
			slog.Info("minio bucket ready", "bucket", cfg.MinioBucket)
		}
	}

	// ---- 依赖注入：Repo 层 ----
	userRepo := repo.NewUserRepo(db)
	emailVerificationRepo := repo.NewEmailVerificationRepo(db)
	refreshTokenRepo := repo.NewRefreshTokenRepo(db)
	passwordResetRepo := repo.NewPasswordResetRepo(db)
	roomRepo := repo.NewRoomRepo(db)
	roomPlayerRepo := repo.NewRoomPlayerRepo(db)
	romRepo := repo.NewRomRepo(db)
	friendRepo := repo.NewFriendRepo(db)

	// ---- 依赖注入：Cache 层 ----
	captchaCache := cache.NewCaptcha(rds)
	roomStateCache := cache.NewRoomState(rds)
	limiterCache := cache.NewLimiter(rds)
	workerRegistry := cache.NewWorkerRegistry(workerRds)

	// ---- 依赖注入：Scheduler + gRPC Client ----
	workerScheduler := scheduler.New()
	workerClient := grpcclient.NewWorkerClient(cfg.WorkerGRPCTimeout)

	// 初始化滑块验证码生成器
	// 优先加载 assets/captcha/ 目录下的自定义背景图，回退到程序化生成的多色渐变背景
	bgImgs, err := captcha.LoadBackgrounds("assets/captcha")
	if err != nil || len(bgImgs) == 0 {
		slog.Error("missing captcha images", "error", err)
		os.Exit(1)
	}

	graphImgs, err := captcha.LoadGraphImages("assets/captcha/tile")
	if err != nil {
		slog.Error("missing captcha tile images", "error", err)
		os.Exit(1)
	}

	slideBuilder := slide.NewBuilder(
		slide.WithImageSize(option.Size{Width: captcha.ImageWidth, Height: captcha.ImageHeight}),
	)
	slideBuilder.SetResources(
		slide.WithBackgrounds(bgImgs),
		slide.WithGraphImages(graphImgs),
	)
	slideCaptcha := slideBuilder.Make()

	// ---- 依赖注入：Service 层 ----

	var emailSender contract.EmailSender
	if cfg.SMTPHost != "" {
		emailSender = email.NewSMTPSender(
			cfg.SMTPHost, cfg.SMTPPort,
			cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom,
			cfg.SMTPUseTLS,
		)
	} else {
		slog.Warn("SMTP not configured, email sending disabled (use token from debug logs)")
		emailSender = &email.NoopSender{}
	}

	authSvc := service.NewAuthService(
		userRepo, emailVerificationRepo, refreshTokenRepo,
		passwordResetRepo,
		captchaCache, slideCaptcha, limiterCache, cfg.JWTSecret,
		minioAdapter, cfg.MinioBucket,
		emailSender, cfg.FrontendBaseURL,
	)
	roomSvc := service.NewRoomService(
		roomRepo, roomPlayerRepo, friendRepo, roomStateCache,
		romRepo, workerScheduler, workerRegistry, workerClient,
		minioAdapter, cfg.MinioBucket,
	)
	romSvc := service.NewRomService(romRepo, minioAdapter, cfg.MinioBucket)
	friendSvc := service.NewFriendService(friendRepo, userRepo)

	// ---- 依赖注入：Handler 层 ----
	handlers := &router.Handlers{
		Auth:   handler.NewAuthHandler(authSvc),
		Room:   handler.NewRoomHandler(roomSvc),
		Rom:    handler.NewRomHandler(romSvc),
		Friend: handler.NewFriendHandler(friendSvc),
		Files:  handler.NewFileHandler(minioAdapter, cfg),
	}

	// 注册路由并启动服务
	r := router.New(cfg, handlers)

	slog.Info("control-plane starting", "addr", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
