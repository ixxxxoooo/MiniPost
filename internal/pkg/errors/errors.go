package errors

import "fmt"

// AppError 统一错误结构，前端可直接解析
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Detail: err.Error()}
}
