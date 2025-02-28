package types

import "time"

// APIResponse는 모든 API 응답에 대한 표준 구조입니다
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Time    string      `json:"time"`
}

// NewAPIResponse는 새로운 API 응답을 생성합니다
func NewAPIResponse(success bool, message string, data interface{}, err string) APIResponse {
	return APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   err,
		Time:    time.Now().Format(time.RFC3339),
	}
}
