package types

type RecordStatus struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (s RecordStatus) String() string {
	return enums[s.Code]
}

var enums = map[int]string{
	-999: "未知异常",
	-400: "代码抛异常",
	-300: "参数不合法",
	-200: "sql执行失败",
	-100: "鉴权失败",
	-1:   "业务代码执行失败",
	0:    "执行成功",
}

var RecordStatusUnknownError = RecordStatus{-999, "未知异常"}
var RecordStatusCodeThrowError = RecordStatus{-400, "代码抛异常"}
var RecordStatusParamIllegalError = RecordStatus{-300, "参数不合法"}
var RecordStatusSqlExecuteError = RecordStatus{-200, "sql执行失败"}
var RecordStatusAuthError = RecordStatus{-100, "鉴权失败"}
var RecordStatusBusinessExecuteError = RecordStatus{-1, "业务代码执行失败"}
var RecordStatusSuccess = RecordStatus{0, "执行成功"}
