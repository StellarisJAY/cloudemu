package router

import (
	"log/slog"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/handler"
	"github.com/StellarisJAY/cloudemu/internal/pkg/config"
	"github.com/StellarisJAY/cloudemu/internal/pkg/logging"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Handlers 聚合所有 HTTP Handler，用于路由注册
type Handlers struct {
	Auth     *handler.AuthHandler
	Room     *handler.RoomHandler
	Rom      *handler.RomHandler
	Admin    *handler.AdminHandler
	Friend   *handler.FriendHandler
	Files    *handler.FileHandler
	UserRepo contract.UserRepo // 供 AdminAuth 中间件查库校验 is_admin
}

// New 创建 gin.Engine 并注册所有路由
// 路由分为三组：
//  1. 公开接口 — 无需登录（captcha、register、login、verify-email、refresh）
//  2. 需登录接口 — 通过 JWTAuth 中间件保护
//  3. 文件代理 — GET /api/files/*path
func New(cfg *config.Config, h *Handlers) *gin.Engine {
	// 注册自定义 validator（notnil_uuid 等），确保 DTO binding 生效
	if err := RegisterValidators(); err != nil {
		slog.Warn("register validators failed", "error", err)
	}

	r := gin.New()

	// 使用 slog 自定义日志中间件替代 gin 默认 Logger
	r.Use(logging.GinLogger(slog.Default()))

	// Recovery 中间件（panic 自动恢复）
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{ // CORS 配置，允许前端开发服务器跨域访问
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		// 公开接口：无需 JWT 认证
		api.GET("/auth/captcha", h.Auth.Captcha)
		api.POST("/auth/captcha/verify", h.Auth.VerifyCaptcha)
		api.POST("/auth/register", h.Auth.Register)
		api.POST("/auth/verify-email", h.Auth.VerifyEmail)
		api.POST("/auth/login", h.Auth.Login)
		api.POST("/auth/resend-code", h.Auth.ResendCode)
		api.POST("/auth/refresh", h.Auth.RefreshToken)
		api.POST("/auth/forgot-password", h.Auth.ForgotPassword)
		api.POST("/auth/reset-password", h.Auth.ResetPassword)

		// 需要 JWT 认证的接口
		auth := api.Group("", JWTAuth(cfg.JWTSecret))
		{
			// 用户信息
			auth.GET("/auth/me", h.Auth.Me)
			auth.PUT("/auth/profile", h.Auth.UpdateProfile)
			auth.PUT("/auth/password", h.Auth.UpdatePassword)

			// 用户搜索
			auth.GET("/users/search", h.Auth.Search)
			auth.GET("/users/:id", h.Auth.GetUser)

			// 好友
			auth.GET("/friends", h.Friend.List)
			auth.GET("/friends/pending", h.Friend.Pending)
			auth.POST("/friends/add", h.Friend.Add)
			auth.POST("/friends/accept", h.Friend.Accept)
			auth.POST("/friends/reject", h.Friend.Reject)

			// 房间
			auth.GET("/rooms", h.Room.List)
			auth.POST("/rooms/create", h.Room.Create)
			auth.POST("/rooms/invite", h.Room.InviteToRoom)
			auth.POST("/rooms/change-role", h.Room.ChangeRole)
			auth.POST("/rooms/select-rom", h.Room.SelectRom)
			auth.POST("/rooms/start", h.Room.Start)
			auth.GET("/rooms/:id/members", h.Room.GetMembers)
			auth.GET("/rooms/:id/livekit", h.Room.GetLivekitToken)
			auth.POST("/rooms/kick", h.Room.KickPlayer)
			auth.POST("/rooms/leave", h.Room.Leave)
			auth.POST("/rooms/pause", h.Room.Pause)
			auth.POST("/rooms/resume", h.Room.Resume)
			auth.POST("/rooms/stop", h.Room.Stop)
			auth.POST("/rooms/delete", h.Room.Delete)
			auth.POST("/rooms/save-state", h.Room.SaveState)
			auth.POST("/rooms/load-state", h.Room.LoadState)
			auth.POST("/rooms/load-latest-state", h.Room.LoadLatestState)
			auth.POST("/rooms/rename-save-state", h.Room.RenameSaveState)
			auth.POST("/rooms/delete-save-state", h.Room.DeleteSaveState)
			auth.GET("/rooms/:id/save-states", h.Room.ListSaveStates)

			// ROM
			auth.GET("/roms", h.Rom.List)
			auth.POST("/roms/upload", h.Rom.Upload)
			auth.PUT("/roms/:id", h.Rom.Update)

			// 管理员：平台内置 ROM 管理（JWTAuth + AdminAuth 查库校验）
			admin := auth.Group("/admin", AdminAuth(h.UserRepo))
			{
				admin.GET("/roms", h.Admin.ListBuiltin)
				admin.POST("/roms/upload", h.Admin.UploadBuiltin)
				admin.PUT("/roms/:id", h.Admin.UpdateBuiltin)
				admin.DELETE("/roms/:id", h.Admin.DeleteBuiltin)
			}
		}

		// MinIO 文件代理
		api.GET("/files/*path", h.Files.Proxy)
	}

	return r
}
