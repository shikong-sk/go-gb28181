package manscdp

import (
	"encoding/xml"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
	"github.com/duke-git/lancet/v2/datastructure/optional"
	"strconv"
)

// CatalogReq 定义了查询目录的请求结构
type CatalogReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

func NewCatalogReq(cmdType, sn, deviceID string) *CatalogReq {
	return &CatalogReq{
		XMLName:  xml.Name{Local: "Query"},
		CmdType:  cmdtype.Catalog,
		SN:       sn,
		DeviceID: deviceID,
	}
}

// CatalogResp 定义了查询目录的响应结构
type CatalogResp struct {
	XMLName    xml.Name           `xml:"Response"`
	CmdType    string             `xml:"CmdType"`
	SumNum     string             `xml:"SumNum"`
	DeviceList *CateLogDeviceList `xml:"DeviceList"`
	SN         string             `xml:"SN"`
	DeviceID   string             `xml:"DeviceID"`
}

// NewCatalogResp 创建一个新的 CatalogResp 实例
func NewCatalogResp(sumNum int, deviceList *CateLogDeviceList, sn, deviceID string) *CatalogResp {
	return &CatalogResp{
		XMLName:    xml.Name{Local: "Response"},
		CmdType:    cmdtype.Catalog,
		SumNum:     strconv.Itoa(sumNum),
		DeviceList: deviceList,
		SN:         sn,
		DeviceID:   deviceID,
	}
}

// CateLogDeviceList 定义了设备列表的结构
type CateLogDeviceList struct {
	XMLName xml.Name        `xml:"DeviceList"`
	Num     string          `xml:"Num,attr"`
	Item    []CateLogDevice `xml:"Item"`
}

// NewCateLogDeviceList 创建一个新的 CateLogDeviceList 实例
func NewCateLogDeviceList(items []CateLogDevice) *CateLogDeviceList {
	return &CateLogDeviceList{
		XMLName: xml.Name{Local: "DeviceList"},
		Num:     strconv.Itoa(len(optional.Of(items).OrElse([]CateLogDevice{}))),
		Item:    items,
	}
}

// CateLogDevice 定义了单个设备的详细信息结构
type CateLogDevice struct {
	XMLName      xml.Name `xml:"Item"`
	Name         string   `xml:"Name"`
	Manufacturer string   `xml:"Manufacturer"`
	Model        string   `xml:"Model"`
	Owner        string   `xml:"Owner"`
	Block        string   `xml:"Block"`
	Address      string   `xml:"Address"`
	Parental     string   `xml:"Parental"`
	SafetyWay    string   `xml:"SafetyWay"`
	RegisterWay  string   `xml:"RegisterWay"`
	CertNum      string   `xml:"CertNum"`
	Certifiable  string   `xml:"Certifiable"`
	ErrCode      string   `xml:"ErrCode"`
	EndTime      string   `xml:"EndTime"`
	Secrecy      string   `xml:"Secrecy"`
	Port         string   `xml:"Port"`
	Password     string   `xml:"Password"`
	Status       string   `xml:"Status"`
	Longitude    string   `xml:"Longitude"`
	Latitude     string   `xml:"Latitude"`
	DeviceID     string   `xml:"DeviceID"`
	CivilCode    string   `xml:"CivilCode"`
	ParentID     string   `xml:"ParentID"`
	IPAddress    string   `xml:"IPAddress"`
}

// NewCateLogDevice 创建一个新的 CateLogDevice 实例，并允许通过传入的函数修改初始值
func NewCateLogDevice(modifier func(*CateLogDevice)) *CateLogDevice {
	device := &CateLogDevice{
		XMLName:      xml.Name{Local: "Item"},
		Name:         "",
		Manufacturer: "",
		Model:        "",
		Owner:        "",
		Block:        "",
		Address:      "",
		Parental:     "",
		SafetyWay:    "",
		RegisterWay:  "",
		CertNum:      "",
		Certifiable:  "",
		ErrCode:      "",
		EndTime:      "",
		Secrecy:      "",
		Port:         "",
		Password:     "",
		Status:       "",
		Longitude:    "",
		Latitude:     "",
		DeviceID:     "",
		CivilCode:    "",
		ParentID:     "",
		IPAddress:    "",
	}
	if modifier != nil {
		modifier(device)
	}
	return device
}
