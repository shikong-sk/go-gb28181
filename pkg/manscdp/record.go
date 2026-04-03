package manscdp

import (
	"encoding/xml"
	"strconv"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

const recordTimeLayout = "2006-01-02T15:04:05"

// RecordInfoReq 历史录像查询请求
// DeviceID 按 GB28181 约定填写通道 ID
// StartTime / EndTime 使用本地时间格式，不带时区
// Secrecy 默认 0，Type 默认 all
// 参考 Java 版本 RecordInfoRequestDTO
// Query 根节点用于平台向设备发起查询
// CmdType 固定为 RecordInfo
type RecordInfoReq struct {
	XMLName   xml.Name `xml:"Query"`
	CmdType   string   `xml:"CmdType"`
	SN        string   `xml:"SN"`
	DeviceID  string   `xml:"DeviceID"`
	StartTime string   `xml:"StartTime"`
	EndTime   string   `xml:"EndTime"`
	Secrecy   string   `xml:"Secrecy"`
	Type      string   `xml:"Type"`
}

func NewRecordInfoReq(sn, channelID string, startTime, endTime time.Time) *RecordInfoReq {
	return &RecordInfoReq{
		XMLName:   xml.Name{Local: "Query"},
		CmdType:   cmdtype.RecordInfo,
		SN:        sn,
		DeviceID:  channelID,
		StartTime: startTime.Format(recordTimeLayout),
		EndTime:   endTime.Format(recordTimeLayout),
		Secrecy:   "0",
		Type:      "all",
	}
}

// RecordInfoResp 历史录像查询响应
// DeviceID 为通道 ID，通常不是上级设备 ID
type RecordInfoResp struct {
	XMLName    xml.Name        `xml:"Response"`
	CmdType    string          `xml:"CmdType"`
	SN         string          `xml:"SN"`
	DeviceID   string          `xml:"DeviceID"`
	Name       string          `xml:"Name"`
	SumNum     string          `xml:"SumNum"`
	RecordList *RecordInfoList `xml:"RecordList"`
}

func (r *RecordInfoResp) Total() int {
	total, _ := strconv.Atoi(r.SumNum)
	return total
}

// RecordInfoList 历史录像列表
// Num 为当前包中的记录数量
type RecordInfoList struct {
	XMLName xml.Name         `xml:"RecordList"`
	Num     string           `xml:"Num,attr"`
	Item    []RecordInfoItem `xml:"Item"`
}

// RecordInfoItem 单条录像信息
// 字段命名与 Java 版本 RecordInfoItemDTO / VO 保持一致
type RecordInfoItem struct {
	XMLName   xml.Name `xml:"Item"`
	DeviceID  string   `xml:"DeviceID"`
	Name      string   `xml:"Name"`
	Address   string   `xml:"Address"`
	StartTime string   `xml:"StartTime"`
	EndTime   string   `xml:"EndTime"`
	Secrecy   string   `xml:"Secrecy"`
	Type      string   `xml:"Type"`
	FileSize  string   `xml:"FileSize"`
}
