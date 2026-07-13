package response

import (
	"errors"
	"net/http"

	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
)

// Body 统一 API 响应结构体，所有接口通过此结构返回
// Code=0 表示成功，非0表示错误（错误码按模块分段）
type Body struct {
	Code    int         `json:"code"`           // 业务状态码，0=成功
	Message string      `json:"message"`        // 状态信息
	Data    interface{} `json:"data,omitempty"` // 响应数据，错误时省略
}

// OK 返回 200 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// Created 返回 201 创建成功响应（通常用于 POST 创建资源后）
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{Code: 0, Message: "created", Data: data})
}

// NoContent 返回 204 无内容响应
func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, Body{Code: 0, Message: "ok"})
}

// Error 统一错误处理：AppError 按其 HTTP 状态码返回，未知错误返回 500
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus(), Body{Code: appErr.Code, Message: appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{Code: 5000, Message: "服务器内部错误"})
}

// BadRequest 返回 400 参数错误
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Body{Code: 4000, Message: msg})
}

// Unauthorized 返回 401 未认证
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Body{Code: 4001, Message: msg})
}
