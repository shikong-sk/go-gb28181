package types

type GetMediaListReq struct {
	Schema string `json:"schema"`
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
}

type MediaOriginSock struct {
	Identifier string `json:"identifier"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
	PeerIP     string `json:"peer_ip"`
	PeerPort   int    `json:"peer_port"`
}

type MediaTrack struct {
	Channels      int    `json:"channels"`
	CodecId       int    `json:"codec_id"`
	CodecIdName   string `json:"codec_id_name"`
	CodecType     int    `json:"codec_type"`
	Fps           int    `json:"fps"`
	Frames        int64  `json:"frames"`
	GopIntervalMs int64  `json:"gop_interval_ms"`
	GopSize       int64  `json:"gop_size"`
	KeyFrames     int64  `json:"key_frames"`
	Ready         bool   `json:"ready"`
	Height        int    `json:"height"`
	Width         int    `json:"width"`
	SampleBit     int    `json:"sample_bit"`
	SampleRate    int    `json:"sample_rate"`
}

type GetMediaListResp struct {
	Schema string `json:"schema"`
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`

	ReaderCount      int64 `json:"readerCount"`
	TotalReaderCount int64 `json:"totalReaderCount"`

	OriginSock MediaOriginSock `json:"originSock"`

	OriginType    int    `json:"originType"`
	OriginTypeStr string `json:"originTypeStr"`
	OriginUrl     string `json:"originUrl"`

	CreateStamp int64 `json:"createStamp"`
	AliveSecond int64 `json:"aliveSecond"`
	BytesSpeed  int64 `json:"bytesSpeed"`

	Tracks []MediaTrack `json:"tracks"`
}
