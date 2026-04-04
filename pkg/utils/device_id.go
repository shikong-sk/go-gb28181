package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DeviceIDParts 设备 ID 解析结果
type DeviceIDParts struct {
	CivilCode    string // 行政区划代码（前8位）
	IndustryCode string // 行业编码（9-10位）
	TypeCode     string // 类型编码（11-13位）
	SerialNumber string // 序号（14-20位）
}

// ValidateDeviceID 校验设备 ID 格式
// 返回：valid（是否有效），warnings（警告信息列表），err（解析错误）
// 兼容性处理：非标准设备ID仅记录警告，不拒绝注册
func ValidateDeviceID(deviceID string) (valid bool, warnings []string, err error) {
	warnings = make([]string, 0)

	// 基础检查：长度和字符
	if deviceID == "" {
		return false, warnings, fmt.Errorf("设备 ID 为空")
	}

	// GB28181 标准：20位编码
	if len(deviceID) != 20 {
		warnings = append(warnings, fmt.Sprintf("设备 ID 长度非标准：期望20位，实际%d位", len(deviceID)))
		// 兼容性：长度不符合标准但仍继续校验
	}

	// 检查是否为纯数字（标准要求）
	if !isAllDigits(deviceID) {
		warnings = append(warnings, "设备 ID 包含非数字字符（标准要求20位纯数字）")
		// 兼容性：允许非数字字符，某些设备可能使用字母
	}

	// 尝试解析各部分
	parts, parseErr := ParseDeviceID(deviceID)
	if parseErr != nil {
		warnings = append(warnings, fmt.Sprintf("设备 ID 解析失败: %v", parseErr))
		// 兼容性：解析失败但仍视为有效（仅警告）
		return true, warnings, parseErr
	}

	// 校验行政区划代码（前8位）
	if err := validateCivilCode(parts.CivilCode); err != nil {
		warnings = append(warnings, fmt.Sprintf("行政区划代码校验失败: %v", err))
	}

	// 校验行业编码（9-10位）
	if err := validateIndustryCode(parts.IndustryCode); err != nil {
		warnings = append(warnings, fmt.Sprintf("行业编码校验失败: %v", err))
	}

	// 校验类型编码（11-13位）
	if err := validateTypeCode(parts.TypeCode); err != nil {
		warnings = append(warnings, fmt.Sprintf("类型编码校验失败: %v", err))
	}

	// 兼容性：即使有警告也视为有效
	valid = true
	return valid, warnings, nil
}

// ParseDeviceID 解析设备 ID 各部分
// 设备 ID 格式（GB28181-2016）：
// - 前8位：行政区划代码
// - 9-10位：行业编码
// - 11-13位：类型编码
// - 14-20位：序号
func ParseDeviceID(deviceID string) (*DeviceIDParts, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("设备 ID 为空")
	}

	// 标准长度检查
	if len(deviceID) < 13 {
		return nil, fmt.Errorf("设备 ID 长度不足：%d（最小需要13位）", len(deviceID))
	}

	// 非标准长度处理：尝试解析尽可能多的部分
	parts := &DeviceIDParts{}

	// 提取行政区划代码（前8位）
	if len(deviceID) >= 8 {
		parts.CivilCode = deviceID[0:8]
	} else {
		parts.CivilCode = deviceID
		return parts, fmt.Errorf("无法提取完整的行政区划代码")
	}

	// 提取行业编码（9-10位）
	if len(deviceID) >= 10 {
		parts.IndustryCode = deviceID[8:10]
	} else {
		// 长度不足，返回已解析部分
		return parts, fmt.Errorf("无法提取行业编码")
	}

	// 提取类型编码（11-13位）
	if len(deviceID) >= 13 {
		parts.TypeCode = deviceID[10:13]
	} else {
		return parts, fmt.Errorf("无法提取类型编码")
	}

	// 提取序号（14-20位）
	if len(deviceID) >= 20 {
		parts.SerialNumber = deviceID[13:20]
	} else if len(deviceID) > 13 {
		// 非标准长度：使用剩余部分作为序号
		parts.SerialNumber = deviceID[13:]
	} else {
		parts.SerialNumber = ""
	}

	return parts, nil
}

// validateCivilCode 校验行政区划代码（前8位）
// 参考 GB/T 2260 行政区划代码标准
func validateCivilCode(civilCode string) error {
	if civilCode == "" {
		return fmt.Errorf("行政区划代码为空")
	}

	if len(civilCode) != 8 {
		return fmt.Errorf("行政区划代码长度非标准：期望8位，实际%d位", len(civilCode))
	}

	if !isAllDigits(civilCode) {
		return fmt.Errorf("行政区划代码包含非数字字符")
	}

	// 基础校验：前2位为省级代码（11-65）
	provinceCode, err := strconv.Atoi(civilCode[0:2])
	if err != nil {
		return fmt.Errorf("省级代码解析失败: %w", err)
	}

	if provinceCode < 11 || provinceCode > 65 {
		return fmt.Errorf("省级代码超出范围：%d（有效范围11-65）", provinceCode)
	}

	return nil
}

// validateIndustryCode 校验行业编码（9-10位）
// 参考 GB/T 28181-2016 行业编码规范
func validateIndustryCode(industryCode string) error {
	if industryCode == "" {
		return fmt.Errorf("行业编码为空")
	}

	if len(industryCode) != 2 {
		return fmt.Errorf("行业编码长度非标准：期望2位，实际%d位", len(industryCode))
	}

	if !isAllDigits(industryCode) {
		return fmt.Errorf("行业编码包含非数字字符")
	}

	// 行业编码范围（常见值）
	// 00-其他
	// 01-公安
	// 02-交通
	// 03-司法
	// 04-教育
	// 05-卫生
	// 06-环保
	// 07-消防
	// 08-林业
	// 09-水利
	// 10-国土
	// 11-城管
	// 12-旅游
	// 13-安全生产
	// ... 其他行业
	code, err := strconv.Atoi(industryCode)
	if err != nil {
		return fmt.Errorf("行业编码解析失败: %w", err)
	}

	// 基础范围检查（00-99）
	if code < 0 || code > 99 {
		return fmt.Errorf("行业编码超出范围：%d（有效范围00-99）", code)
	}

	// 警告非常见行业编码（但不拒绝）
	if code > 20 {
		return fmt.Errorf("行业编码 %d 为非常见值（常见值00-20）", code)
	}

	return nil
}

// validateTypeCode 校验类型编码（11-13位）
// 参考 GB/T 28181-2016 设备类型编码规范
func validateTypeCode(typeCode string) error {
	if typeCode == "" {
		return fmt.Errorf("类型编码为空")
	}

	if len(typeCode) != 3 {
		return fmt.Errorf("类型编码长度非标准：期望3位，实际%d位", len(typeCode))
	}

	if !isAllDigits(typeCode) {
		return fmt.Errorf("类型编码包含非数字字符")
	}

	// 设备类型编码范围
	// 100-199：虚拟组织/系统
	// 200-299：编码设备
	// 300-399：解码设备
	// 400-499：转换设备
	// 500-599：服务器
	// 600-699：客户端
	// 700-799：显示设备
	// 800-899：网络传输设备
	// 900-999：其他设备
	code, err := strconv.Atoi(typeCode)
	if err != nil {
		return fmt.Errorf("类型编码解析失败: %w", err)
	}

	if code < 100 || code > 999 {
		return fmt.Errorf("类型编码超出范围：%d（有效范围100-999）", code)
	}

	// 常见类型检查（警告但不拒绝）
	// 111-119：虚拟组织
	// 121-129：业务平台
	// 131-139：摄像头
	// 141-149：编码器
	// 等等...
	isCommonType := (code >= 111 && code <= 119) ||
		(code >= 121 && code <= 129) ||
		(code >= 131 && code <= 139) ||
		(code >= 141 && code <= 149) ||
		(code >= 151 && code <= 159) ||
		(code >= 200 && code <= 299) ||
		(code >= 300 && code <= 399)

	if !isCommonType {
		return fmt.Errorf("类型编码 %d 为非常见值", code)
	}

	return nil
}

// isAllDigits 检查字符串是否为纯数字
func isAllDigits(s string) bool {
	matched, _ := regexp.MatchString("^\\d+$", s)
	return matched
}

// FormatDeviceIDWarnings 格式化设备 ID 校验警告信息
func FormatDeviceIDWarnings(deviceID string, warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}

	return fmt.Sprintf("设备 ID %s 校验警告：%s", deviceID, strings.Join(warnings, "; "))
}
