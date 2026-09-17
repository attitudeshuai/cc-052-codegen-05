package service

// BizError 业务校验错误,handler 按 Status 返回对应 HTTP 状态码
type BizError struct {
	Status  int
	Message string
}

func (e *BizError) Error() string { return e.Message }

func newBizError(status int, message string) *BizError {
	return &BizError{Status: status, Message: message}
}
