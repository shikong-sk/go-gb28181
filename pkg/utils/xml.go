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

	switch csStr {
	case "", cs.UTF8:
		return []byte(xmlStr), nil
	case cs.GBK, cs.GB2312:
		return convertUTF8ToGBK([]byte(xmlStr))
	case cs.GB18030:
		return convertUTF8ToGB18030([]byte(xmlStr))
	default:
		enc := mahonia.NewEncoder(charset)
		if enc == nil {
			return nil, fmt.Errorf("不支持的 XML 编码: %s", charset)
		}
		return []byte(enc.ConvertString(xmlStr)), nil
	}
}

// XMLUnmarshal 将XML格式的字节数组data反序列化为对象obj
// 自动检测编码（GBK/GB2312/UTF-8）并进行转换
func XMLUnmarshal(data []byte, obj interface{}) error {
	// 1. 首先尝试使用标准方法（依赖 XML 声明的 encoding）
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		reader, err := charset.NewReader(input, c)
		return reader, err
	}

	err := decoder.Decode(&obj)
	if err == nil {
		return nil
	}

	// 2. 如果标准方法失败，尝试检测编码并转换
	utf8Data, convErr := DetectAndConvertToUTF8(data)
	if convErr != nil {
		return fmt.Errorf("XML解析失败且编码转换失败: %w (原始错误: %w)", convErr, err)
	}

	// 3. 使用转换后的 UTF-8 数据重新解析
	decoder = xml.NewDecoder(bytes.NewReader(utf8Data))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		reader, err := charset.NewReader(input, c)
		return reader, err
	}

	return decoder.Decode(&obj)
}

// DetectAndConvertToUTF8 检测编码并转换为 UTF-8
func DetectAndConvertToUTF8(data []byte) ([]byte, error) {
	// 检测编码
	encoding := detectEncoding(data)

	switch strings.ToUpper(encoding) {
	case cs.GBK, cs.GB2312:
		return convertGBKToUTF8(data)
	case cs.GB18030:
		return convertGB18030ToUTF8(data)
	default:
		// 假设已经是 UTF-8，验证一下
		if isValidUTF8(data) {
			return data, nil
		}
		// 如果不是有效的 UTF-8，尝试 GBK
		return convertGBKToUTF8(data)
	}
}

// detectEncoding 检测数据编码
func detectEncoding(data []byte) string {
	// 1. 检查 XML 声明中的 encoding
	declEncoding := extractXMLEncoding(data)
	if declEncoding != "" {
		return declEncoding
	}

	// 2. 检查字节顺序标记 (BOM)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return cs.UTF8
	}

	// 3. 检查是否为有效的 UTF-8
	if isValidUTF8(data) {
		return cs.UTF8
	}

	// 4. 默认假设为 GBK（GB28181 设备常用）
	return cs.GBK
}

// extractXMLEncoding 从 XML 声明中提取 encoding
func extractXMLEncoding(data []byte) string {
	// 查找 XML 声明
	dataStr := string(data[:min(200, len(data))])

	// 匹配 encoding="xxx" 或 encoding='xxx'
	idx := strings.Index(strings.ToLower(dataStr), "encoding=")
	if idx == -1 {
		return ""
	}

	rest := dataStr[idx+9:] // len("encoding=")

	// 跳过空白
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return ""
	}

	// 获取引号类型
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return ""
	}

	// 查找结束引号
	endIdx := strings.IndexByte(rest[1:], quote)
	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(rest[1 : endIdx+1])
}

// isValidUTF8 检查是否为有效的 UTF-8
func isValidUTF8(data []byte) bool {
	// 简单检查：尝试转换为 string，检查是否有无效字符
	for i := 0; i < len(data); {
		r, size := rune(data[i]), 1
		if data[i] >= 0x80 {
			// 多字节字符
			if data[i] >= 0xC0 && data[i] < 0xE0 && i+1 < len(data) {
				r = rune(data[i]&0x1F)<<6 | rune(data[i+1]&0x3F)
				size = 2
			} else if data[i] >= 0xE0 && data[i] < 0xF0 && i+2 < len(data) {
				r = rune(data[i]&0x0F)<<12 | rune(data[i+1]&0x3F)<<6 | rune(data[i+2]&0x3F)
				size = 3
			} else if data[i] >= 0xF0 && data[i] < 0xF8 && i+3 < len(data) {
				r = rune(data[i]&0x07)<<18 | rune(data[i+1]&0x3F)<<12 | rune(data[i+2]&0x3F)<<6 | rune(data[i+3]&0x3F)
				size = 4
			} else {
				// 无效的 UTF-8 起始字节
				return false
			}
		}
		if r == 0xFFFD {
			return false
		}
		i += size
	}
	return true
}

// convertGBKToUTF8 将 GBK 转换为 UTF-8
func convertGBKToUTF8(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	return io.ReadAll(reader)
}

// convertGB18030ToUTF8 将 GB18030 转换为 UTF-8
func convertGB18030ToUTF8(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	return io.ReadAll(reader)
}

// convertUTF8ToGBK 将 UTF-8 转换为 GBK
func convertUTF8ToGBK(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewEncoder())
	return io.ReadAll(reader)
}

// convertUTF8ToGB18030 将 UTF-8 转换为 GB18030
func convertUTF8ToGB18030(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewEncoder())
	return io.ReadAll(reader)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
