package router

import (
	"strings"

	jwtutil "github.com/StellarisJAY/cloudemu/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
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
