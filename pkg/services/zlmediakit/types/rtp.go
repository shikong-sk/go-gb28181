package types

type OpenRtpServerReq struct {
	Port      int    `json:"port"`
	TcpMode   int    `json:"tcp_mode"`
	StreamId  string `json:"stream_id"`
	SsrcCheck bool   `json:"ssrc_check"` // 是否校验 SSRC，WVP 设置为 false
}

type OpenRtpServerResp struct {
	Code int `json:"code"`
	Port int `json:"port"`
}

type CloseRtpServerReq struct {
	StreamId string `json:"stream_id"`
}

type CloseRtpServerResp struct {
	Code int `json:"code"`
	Hit  int `json:"hit"`
}

type ConnectRtpServerReq struct {
	StreamId string `json:"stream_id"`
	DstUrl   string `json:"dst_url"`
	DstPort  int    `json:"dst_port"`
}

type ConnectRtpServerResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type StartSendRtpReq struct {
	Vhost     string `json:"vhost"`
	App       string `json:"app"`
	Stream    string `json:"stream"`
	Ssrc      string `json:"ssrc"`
	DstUrl    string `json:"dst_url"`
	DstPort   int    `json:"dst_port"`
	IsUdp     bool   `json:"is_udp"`
	SrcPort   int    `json:"src_port"`
	Pt        int    `json:"pt"`
	UsePs     bool   `json:"use_ps"`
	OnlyAudio bool   `json:"only_audio"`
}

type StartSendRtpPassiveReq struct {
	Vhost     string `json:"vhost"`
	App       string `json:"app"`
	Stream    string `json:"stream"`
	Ssrc      string `json:"ssrc"`
	SrcPort   int    `json:"src_port"`
	Pt        int    `json:"pt"`
	UsePs     bool   `json:"use_ps"`
	OnlyAudio bool   `json:"only_audio"`
}

type StartSendRtpResp struct {
	Code      int `json:"code"`
	LocalPort int `json:"local_port"`
}

type StopSendRtpReq struct {
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	Ssrc   string `json:"ssrc"`
}

type RtpServer struct {
	Port     int    `json:"port"`
	StreamId string `json:"stream_id"`
}

type ListRtpServerResp = Data[[]RtpServer]

// RtpInfoResp RTP 流信息响应
type RtpInfoResp struct {
	Code  int    `json:"code"`
	Exist bool   `json:"exist"` // 流是否存在
	IP    string `json:"ip"`    // 推流 IP
	Port  int    `json:"port"`  // 推流端口
}
