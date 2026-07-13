package router

import (
	"reflect"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// RegisterValidators 向 gin 默认 validator 注册自定义校验规则
// 当前注册：
//   - notnil_uuid: 校验 uuid.UUID / *uuid.UUID 字段非空且非 uuid.Nil
//     用于 DTO 中所有必填的 ID 字段，防止前端传入 "00000000-0000-0000-0000-000000000000" 绕过 required 校验
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	return v.RegisterValidation("notnil_uuid", notNilUUID)
}

// notNilUUID 校验函数：字段不能为 nil 指针，且 UUID 值不能为全零
func notNilUUID(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return false
		}
		if u, ok := field.Elem().Interface().(uuid.UUID); ok {
			return u != uuid.Nil
		}
		return false
	}
	if u, ok := field.Interface().(uuid.UUID); ok {
		return u != uuid.Nil
	}
	return false
}
