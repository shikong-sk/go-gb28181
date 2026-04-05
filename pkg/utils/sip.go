package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

const (
	// DefaultAlphabet 用于生成 NanoId 风格的随机字符串
	DefaultAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	// ViaTagPrefix SIP Via 分支标签前缀 (RFC 3261)
	ViaTagPrefix = "z9hG4bK"
)

// GenerateNanoId 生成指定长度的随机字符串 (类似 Java 的 NanoId)
func GenerateNanoId(size int) string {
	result := make([]byte, size)
	alphabetLen := big.NewInt(int64(len(DefaultAlphabet)))
	for i := 0; i < size; i++ {
		n, _ := rand.Int(rand.Reader, alphabetLen)
		result[i] = DefaultAlphabet[n.Int64()]
	}
	return string(result)
}

// GenerateFromTag 生成 SIP From 头的 tag 参数
// 使用 32 位随机字符串 (与 Java 版本一致)
func GenerateFromTag() string {
	return GenerateNanoId(32)
}

// GenerateViaTag 生成 SIP Via 头的 branch 参数
// 格式: z9hG4bK + 10位随机字符 (RFC 3261)
func GenerateViaTag() string {
	return ViaTagPrefix + GenerateNanoId(10)
}

// GenerateTag 生成基于时间戳的 tag
func GenerateTag() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// GenerateSN 生成 6 位随机序号
func GenerateSN() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("%06d", n.Int64()+1)
}

// GenerateCallID 生成 SIP Call-ID（WVP 兼容格式）
// 格式: 32位随机字符串（用于 @ 前缀）
func GenerateCallID() string {
	return GenerateNanoId(32)
}
