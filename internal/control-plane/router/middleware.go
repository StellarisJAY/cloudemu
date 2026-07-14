package router

import (
	"strings"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	jwtutil "github.com/StellarisJAY/cloudemu/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// JWTAuth JWT 认证中间件
// 从 Authorization header 提取 Bearer token，解析后将 user_id 和 username 注入 gin.Context
// 解析失败返回 401
func JWTAuth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		if tokenStr == "" {
			c.AbortWithStatusJSON(401, map[string]interface{}{
				"code":    4001,
				"message": "未登录",
			})
			return
		}

		claims, err := jwtutil.Parse(tokenStr, secret)
		if err != nil {
			c.AbortWithStatusJSON(401, map[string]interface{}{
				"code":    4001,
				"message": "token 无效或已过期",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// AdminAuth 管理员权限中间件
// 必须在 JWTAuth 之后使用：读取 user_id，查库校验 users.is_admin
// 非管理员返回 403（错误码 1014）。权限改动实时生效，无需重新登录
func AdminAuth(userRepo contract.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(uuid.UUID)

		user, err := userRepo.ByID(c.Request.Context(), userID)
		if err != nil || user == nil || !user.IsAdmin {
			c.AbortWithStatusJSON(403, map[string]interface{}{
				"code":    1014,
				"message": "需要管理员权限",
			})
			return
		}
		c.Next()
	}
}
