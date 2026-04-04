package hook

// HookRequest ZLM Hook 请求基础结构
type HookRequest struct {
	MediaServerId string `json:"mediaServerId"` // 流媒体服务器 ID
	App           string `json:"app"`           // 应用名
	Stream        string `json:"stream"`        // 流 ID
	Schema        string `json:"schema"`        // 协议类型
	Vhost         string `json:"vhost"`         // 虚拟主机
	Params        string `json:"params"`        // URL 参数
}

// StreamChangedRequest 流状态变化请求
type StreamChangedRequest struct {
	HookRequest
	Register bool `json:"register"` // true=注册(流创建), false=注销(流销毁)
}

// StreamNoneReaderRequest 无观看者请求
type StreamNoneReaderRequest struct {
	HookRequest
}

// RtpServerTimeoutRequest RTP 服务器超时请求
type RtpServerTimeoutRequest struct {
	HookRequest
}

// HookResponse Hook 响应
type HookResponse struct {
	Code int    `json:"code"` // 0=允许/接受, -1=拒绝
	Msg  string `json:"msg"`  // 消息
}

// SuccessHookResponse 返回成功的 Hook 响应
func SuccessHookResponse(msg string) *HookResponse {
	return &HookResponse{
		Code: 0,
		Msg:  msg,
	}
}

// RejectHookResponse 返回拒绝的 Hook 响应
func RejectHookResponse(msg string) *HookResponse {
	return &HookResponse{
		Code: -1,
		Msg:  msg,
	}
}
