package types

type Data[T any] struct {
	Code int `json:"code"`
	Data T   `json:"data"`
}
