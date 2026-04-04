package utils

import (
	"fmt"
	"testing"
)

// TestValidateDeviceID_Standard20Digits 测试标准20位纯数字设备ID
func TestValidateDeviceID_Standard20Digits(t *testing.T) {
	// 标准20位纯数字设备ID（示例）
	deviceID := "44050100001310000010"

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("设备ID: %s\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 兼容性要求：即使有警告也应视为有效
	if !valid {
		t.Errorf("标准设备ID应视为有效，但返回valid=false")
	}

	// 可能会有警告（如行业编码或类型编码非常见值），这是正常的
	if len(warnings) > 0 {
		t.Logf("设备ID校验产生%d个警告（这是兼容性处理的正常情况）", len(warnings))
	}
}

// TestValidateDeviceID_19Digits 测试19位设备ID（兼容性）
func TestValidateDeviceID_19Digits(t *testing.T) {
	// 19位设备ID（长度不足）
	deviceID := "4405010000131000001"

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("\n设备ID: %s\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 兼容性要求：长度不符合标准但仍应视为有效
	if !valid {
		t.Errorf("非标准长度设备ID应视为有效（兼容性），但返回valid=false")
	}

	// 应产生长度警告
	hasLengthWarning := false
	for _, w := range warnings {
		if contains(w, "长度") {
			hasLengthWarning = true
			break
		}
	}

	if !hasLengthWarning {
		t.Error("非标准长度设备ID应产生长度警告")
	}
}

// TestValidateDeviceID_WithLetters 测试包含字母的设备ID（兼容性）
func TestValidateDeviceID_WithLetters(t *testing.T) {
	// 包含字母的设备ID
	deviceID := "440501000013100000AB"

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("\n设备ID: %s\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 兼容性要求：包含非数字字符但仍应视为有效
	if !valid {
		t.Errorf("包含字母的设备ID应视为有效（兼容性），但返回valid=false")
	}

	// 应产生非数字字符警告
	hasNonDigitWarning := false
	for _, w := range warnings {
		if contains(w, "非数字") {
			hasNonDigitWarning = true
			break
		}
	}

	if !hasNonDigitWarning {
		t.Error("包含字母的设备ID应产生非数字字符警告")
	}
}

// TestValidateDeviceID_InvalidIndustryCode 测试非常见行业编码
func TestValidateDeviceID_InvalidIndustryCode(t *testing.T) {
	// 行业编码为99（超出常见范围00-20）
	// 格式：行政区划(8位) + 行业编码(2位) + 类型编码(3位) + 序号(7位)
	deviceID := "44050100991310000010"

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("\n设备ID: %s\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 兼容性要求：行业编码非常见但仍应视为有效
	if !valid {
		t.Errorf("非常见行业编码的设备ID应视为有效（兼容性），但返回valid=false")
	}

	// 应产生行业编码警告
	hasIndustryWarning := false
	for _, w := range warnings {
		if contains(w, "行业编码") {
			hasIndustryWarning = true
			break
		}
	}

	if !hasIndustryWarning {
		t.Error("非常见行业编码应产生警告")
	}
}

// TestValidateDeviceID_Empty 测试空设备ID
func TestValidateDeviceID_Empty(t *testing.T) {
	deviceID := ""

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("\n设备ID: %s（空）\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 空设备ID应视为无效
	if valid {
		t.Error("空设备ID应视为无效")
	}

	if err == nil {
		t.Error("空设备ID应返回错误")
	}
}

// TestValidateDeviceID_Short 测试过短设备ID
func TestValidateDeviceID_Short(t *testing.T) {
	// 过短设备ID（不足13位）
	deviceID := "44050100"

	valid, warnings, err := ValidateDeviceID(deviceID)

	fmt.Printf("\n设备ID: %s（过短）\n", deviceID)
	fmt.Printf("有效: %v\n", valid)
	fmt.Printf("警告: %v\n", warnings)
	fmt.Printf("错误: %v\n", err)

	// 兼容性要求：过短设备ID仍应视为有效（仅警告）
	if !valid {
		t.Errorf("过短设备ID应视为有效（兼容性），但返回valid=false")
	}

	// 应产生警告
	if len(warnings) == 0 {
		t.Error("过短设备ID应产生警告")
	}
}

// TestParseDeviceID 测试设备ID解析
func TestParseDeviceID(t *testing.T) {
	deviceID := "44050100001310000010"

	parts, err := ParseDeviceID(deviceID)

	if err != nil {
		t.Errorf("解析标准设备ID失败: %v", err)
	}

	fmt.Printf("\n设备ID解析结果:\n")
	fmt.Printf("行政区划代码: %s\n", parts.CivilCode)
	fmt.Printf("行业编码: %s\n", parts.IndustryCode)
	fmt.Printf("类型编码: %s\n", parts.TypeCode)
	fmt.Printf("序号: %s\n", parts.SerialNumber)

	// 验证各部分
	if parts.CivilCode != "44050100" {
		t.Errorf("行政区划代码解析错误：期望44050100，实际%s", parts.CivilCode)
	}

	if parts.IndustryCode != "00" {
		t.Errorf("行业编码解析错误：期望00，实际%s", parts.IndustryCode)
	}

	if parts.TypeCode != "131" {
		t.Errorf("类型编码解析错误：期望131，实际%s", parts.TypeCode)
	}

	if parts.SerialNumber != "0000010" {
		t.Errorf("序号解析错误：期望0000010，实际%s", parts.SerialNumber)
	}
}

// TestFormatDeviceIDWarnings 测试警告格式化
func TestFormatDeviceIDWarnings(t *testing.T) {
	deviceID := "440501000013100000AB"
	warnings := []string{"设备ID包含非数字字符", "设备ID长度非标准"}

	result := FormatDeviceIDWarnings(deviceID, warnings)

	fmt.Printf("\n格式化警告: %s\n", result)

	if result == "" {
		t.Error("警告格式化结果不应为空")
	}

	if !contains(result, deviceID) {
		t.Error("格式化警告应包含设备ID")
	}
}

// contains 检查字符串是否包含子串（辅助函数）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
