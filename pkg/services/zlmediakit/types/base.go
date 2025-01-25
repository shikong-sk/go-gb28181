package types

// Data 是一个泛型结构体，用于封装 API 响应的数据
type Data[T any] struct {
	Code int    `json:"code"`
	Data T      `json:"data"`
	Msg  string `json:"msg"`
}
