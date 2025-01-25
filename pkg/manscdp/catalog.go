package manscdp

import "encoding/xml"

// CatalogReq 定义了查询目录的请求结构
type CatalogReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
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

// CateLogDeviceList 定义了设备列表的结构
type CateLogDeviceList struct {
	XMLName xml.Name        `xml:"DeviceList"`
	Num     string          `xml:"Num,attr"`
	Item    []CateLogDevice `xml:"Item"`
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
