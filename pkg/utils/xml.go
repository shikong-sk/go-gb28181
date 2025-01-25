package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	cs "git.skcks.cn/Shikong/go-gb28181/pkg/utils/charset"
	"github.com/axgle/mahonia"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"strings"
)

// XMLMarshalString 将给定的对象v序列化为XML字符串，并指定字符集charset
func XMLMarshalString(v interface{}, charset string) (string, error) {
	marshal, err := XMLMarshal(v, charset)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}

// XMLMarshal 将给定的对象obj序列化为XML格式的字节数组，并指定字符集charset
func XMLMarshal(obj interface{}, charset string) ([]byte, error) {
	marshal := &bytes.Buffer{}
	encoder := xml.NewEncoder(marshal)
	encoder.Indent("", " ")
	err := encoder.Encode(&obj)
	if err != nil {
		return nil, err
	}
	err = encoder.Close()
	if err != nil {
		return nil, err
	}

	xmlStr := marshal.String()
	csStr := strings.ToUpper(charset)
	xmlStr = fmt.Sprintf("<?xml version=\"1.0\" encoding=\"%s\" ?>\r\n%s", csStr, xmlStr)

	xmlBytes := &bytes.Buffer{}
	err = func() (err error) {
		defer func() {
			if err := recover(); err != nil {
				var t transform.Transformer
				switch strings.ToUpper(charset) {
				case cs.GBK:
					t = simplifiedchinese.GBK.NewEncoder()
				case cs.GB2312:
					t = simplifiedchinese.HZGB2312.NewEncoder()
				case cs.GB18030:
					t = simplifiedchinese.GB18030.NewEncoder()
				default:
					err = fmt.Errorf("unsupported charset: %s", charset)
					return
				}

				writer := transform.NewWriter(xmlBytes, t)
				_, _ = writer.Write([]byte(xmlStr))
				_ = writer.Close()
			}
		}()

		enc := mahonia.NewEncoder(charset)
		xmlStr := enc.ConvertString(xmlStr)
		xmlBytes.Write([]byte(xmlStr))
		return nil
	}()

	return xmlBytes.Bytes(), nil
}

// XMLUnmarshal 将XML格式的字节数组data反序列化为对象obj
func XMLUnmarshal(data []byte, obj interface{}) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		reader, err := charset.NewReader(input, c)
		return reader, err
	}

	err := decoder.Decode(&obj)
	if err != nil {
		return err
	}

	return nil
}
