package utils

import (
	"bytes"
	"github.com/axgle/mahonia"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"os"
	"path/filepath"
)
import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"
)

const GB2312_XML = `
<?xml version="1.0" encoding="GB2312" ?>
<Response>
  <CmdType>Catalog</CmdType>
  <SumNum>2</SumNum>
  <DeviceList Num="2">
    <Item>
      <Name>&#x6a21;&#x62df;&#x8bbe;&#x5907;&#x540d;&#x79f0;</Name>
      <Manufacturer>gb28181-docking-platform</Manufacturer>
      <Model/>
      <Owner/>
      <Block/>
      <Address/>
      <Parental>0</Parental>
      <SafetyWay>0</SafetyWay>
      <RegisterWay>1</RegisterWay>
      <CertNum/>
      <Certifiable>0</Certifiable>
      <ErrCode>0</ErrCode>
      <EndTime/>
      <Secrecy>0</Secrecy>
      <Port/>
      <Password/>
      <Status>ON</Status>
      <Longitude>0.0</Longitude>
      <Latitude>0.0</Latitude>
      <DeviceID>44050100001310000010</DeviceID>
      <CivilCode/>
      <ParentID/>
      <IPAddress/>
    </Item>
    <Item>
      <Name>&#x6a21;&#x62df;&#x8bbe;&#x5907;&#x540d;&#x79f0;</Name>
      <Manufacturer>gb28181-docking-platform</Manufacturer>
      <Model/>
      <Owner/>
      <Block/>
      <Address/>
      <Parental>0</Parental>
      <SafetyWay>0</SafetyWay>
      <RegisterWay>1</RegisterWay>
      <CertNum/>
      <Certifiable>0</Certifiable>
      <ErrCode>0</ErrCode>
      <EndTime/>
      <Secrecy>0</Secrecy>
      <Port/>
      <Password/>
      <Status>ON</Status>
      <Longitude>0.0</Longitude>
      <Latitude>0.0</Latitude>
      <DeviceID>44050100001310000010</DeviceID>
      <CivilCode/>
      <ParentID/>
      <IPAddress/>
    </Item>
  </DeviceList>
  <SN>1</SN>
  <DeviceID>44050100001110000010</DeviceID>
</Response>
`

type Catalog struct {
	XMLName    xml.Name `xml:"Response"`
	CmdType    string   `xml:"CmdType"`
	SumNum     string   `xml:"SumNum"`
	DeviceList struct {
		XMLName xml.Name `xml:"DeviceList"`
		Num     string   `xml:"Num,attr"`
		Item    []struct {
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
		} `xml:"Item"`
	} `xml:"DeviceList"`
	SN       string `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
}

func TestGB2312XML(t *testing.T) {
	decoder := xml.NewDecoder(strings.NewReader(GB2312_XML))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		reader, err := charset.NewReader(input, c)
		return reader, err
	}

	obj := Catalog{}
	err := decoder.Decode(&obj)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v\n", obj)

	marshal := &bytes.Buffer{}
	encoder := xml.NewEncoder(marshal)
	encoder.Indent("", " ")
	err = encoder.Encode(&obj)
	if err != nil {
		t.Error(err)
	}
	err = encoder.Close()
	if err != nil {
		t.Error(err)
	}

	xmlStr := marshal.String()
	cs := strings.ToUpper("gb2312")
	xmlStr = fmt.Sprintf("<?xml version=\"1.0\" encoding=\"%s\" ?>\r\n%s", cs, xmlStr)

	xmlBytes := &bytes.Buffer{}
	err = func() (err error) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("使用 simplifiedchinese 转义")
				writer := transform.NewWriter(xmlBytes, simplifiedchinese.HZGB2312.NewEncoder())
				writer.Write([]byte(xmlStr))
				_ = writer.Close()
			}
		}()

		enc := mahonia.NewEncoder(cs)
		xmlStr = enc.ConvertString(xmlStr)
		xmlBytes.Write([]byte(xmlStr))
		return nil
	}()
	if err != nil {
		return
	}

	fmt.Println(xmlStr)

	tmpFile, _ := filepath.Abs(filepath.Join(os.TempDir(), "gb2312.xml"))
	fmt.Println(tmpFile)
	openFile, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		t.Error(err)
	}
	_, err = openFile.Write(xmlBytes.Bytes())
	if err != nil {
		t.Error(err)
	}
	_ = openFile.Close()

	readAll, err := io.ReadAll(openFile)
	decoder = xml.NewDecoder(bytes.NewBuffer(readAll))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		reader, err := charset.NewReader(input, c)
		return reader, err
	}
	decoder.Decode(&obj)
	fmt.Printf("%+v\n", obj)
}

func TestXML(t *testing.T) {
	fmt.Printf("%s\n", GB2312_XML)

	obj := Catalog{}
	err := XMLUnmarshal([]byte(GB2312_XML), obj)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", &obj)

	fmt.Println(XMLMarshalString(&obj, "GB2312"))
	fmt.Println(XMLMarshalString(&obj, "GBK"))
	fmt.Println(XMLMarshalString(&obj, "UTF8"))
}
