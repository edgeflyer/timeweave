package auth

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrEmailExists 邮箱已被注册。
	ErrEmailExists = errors.New("auth: email already exists")
	// ErrUserNotFound 用户不存在。
	ErrUserNotFound = errors.New("auth: user not found")
	// ErrWrongPassword 密码错误。
	ErrWrongPassword = errors.New("auth: wrong password")
	// ErrUserDisabled 用户被禁用。
	ErrUserDisabled = errors.New("auth: user disabled")
)