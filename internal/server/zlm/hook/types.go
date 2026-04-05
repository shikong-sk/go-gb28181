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
// 注意：ZLM 文档中字段名是 "regist" 不是 "register"
type StreamChangedRequest struct {
	HookRequest
	Regist           bool   `json:"regist"`           // true=注册(流创建), false=注销(流销毁)
	AliveSecond      int    `json:"aliveSecond"`      // 存活时间，单位秒（注册时提供）
	BytesSpeed       int    `json:"bytesSpeed"`       // 数据产生速度，单位byte/s（注册时提供）
	CreateStamp      int    `json:"createStamp"`      // GMT unix系统时间戳，单位秒（注册时提供）
	OriginType       int    `json:"originType"`       // 产生源类型（注册时提供）
	OriginTypeStr    string `json:"originTypeStr"`    // 产生源类型名称（注册时提供）
	OriginUrl        string `json:"originUrl"`        // 产生源的url（注册时提供）
	ReaderCount      int    `json:"readerCount"`      // 本协议观看人数（注册时提供）
	TotalReaderCount int    `json:"totalReaderCount"` // 观看总人数（注册时提供）
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
