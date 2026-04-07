package manscdp

import (
	"encoding/xml"
	"fmt"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// PTZDirection 云台方向
type PTZDirection string

const (
	PTZStop      PTZDirection = "Stop"      // 停止
	PTZUp        PTZDirection = "Up"        // 上
	PTZDown      PTZDirection = "Down"      // 下
	PTZLeft      PTZDirection = "Left"      // 左
	PTZRight     PTZDirection = "Right"     // 右
	PTZUpLeft    PTZDirection = "UpLeft"    // 上左
	PTZUpRight   PTZDirection = "UpRight"   // 上右
	PTZDownLeft  PTZDirection = "DownLeft"  // 下左
	PTZDownRight PTZDirection = "DownRight" // 下右
	PTZZoomIn    PTZDirection = "ZoomIn"    // 放大
	PTZZoomOut   PTZDirection = "ZoomOut"   // 缩小
	PTZFocusNear PTZDirection = "FocusNear" // 近焦
	PTZFocusFar  PTZDirection = "FocusFar"  // 远焦
)

// DeviceControlReq 设备控制请求 (云台控制)
type DeviceControlReq struct {
	XMLName  xml.Name     `xml:"Control"`
	CmdType  string       `xml:"CmdType"`
	SN       string       `xml:"SN"`
	DeviceID string       `xml:"DeviceID"`
	PTZCmd   *PTZCmd      `xml:"PTZCmd,omitempty"`
	Info     *ControlInfo `xml:"Info,omitempty"`
}

// ControlInfo 控制信息
type ControlInfo struct {
	XMLName         xml.Name `xml:"Info"`
	ControlPriority int      `xml:"ControlPriority"` // 控制优先级，范围 0-10，默认 5
}

// PTZCmd 云台控制命令
type PTZCmd struct {
	XMLName xml.Name `xml:"PTZCmd"`
	Value   string   `xml:",chardata"` // 云台控制命令字符串
}

// NewPTZControlReq 创建云台控制请求
func NewPTZControlReq(sn, deviceID string, ptzCmd string) *DeviceControlReq {
	return &DeviceControlReq{
		XMLName:  xml.Name{Local: "Control"},
		CmdType:  cmdtype.DeviceControl,
		SN:       sn,
		DeviceID: deviceID,
		PTZCmd:   &PTZCmd{Value: ptzCmd},
		Info:     &ControlInfo{ControlPriority: 5},
	}
}

// BuildPTZCmd 构建云台控制命令字符串
// GB28181 Pelco-D 扩展格式 (8字节):
// 字节0-1: 同步头 (A5 0F 固定)
// 字节2: 组合码 (云台控制固定 01)
// 字节3: 命令码 (方向控制位)
// 字节4: 水平速度 (0-255)
// 字节5: 垂直速度 (0-255)
// 字节6: Zoom速度/预留
// 字节7: 校验和 (字节1-6按位异或，不含A5)
//
// 命令码位定义 (根据WVP成功日志验证):
// Bit0 (0x01): 右转/电机2正转
// Bit1 (0x02): 左转/电机2反转
// Bit2 (0x04): 下转/电机1反转
// Bit3 (0x08): 上转/电机1正转
// Bit4 (0x10): 放大
// Bit5 (0x20): 缩小
//
// 示例 (WVP成功日志):
// - 向左: A50F01021E1E1003 (命令码02=左, 水平速度1E=30, 垂直速度1E=30)
// - 停止: A50F0100000000B5 (命令码00=停止, 校验和0F^01=0xB5)
func BuildPTZCmd(direction PTZDirection, horizontalSpeed, verticalSpeed int) string {
	// 命令码映射 (根据WVP日志验证的格式)
	var cmdCode byte
	switch direction {
	case PTZStop:
		cmdCode = 0x00
	case PTZUp:
		cmdCode = 0x08 // Bit3: 上转
	case PTZDown:
		cmdCode = 0x04 // Bit2: 下转
	case PTZLeft:
		cmdCode = 0x02 // Bit1: 左转 (WVP验证)
	case PTZRight:
		cmdCode = 0x01 // Bit0: 右转
	case PTZUpLeft:
		cmdCode = 0x0A // 上(08) + 左(02) = 0A
	case PTZUpRight:
		cmdCode = 0x09 // 上(08) + 右(01) = 09
	case PTZDownLeft:
		cmdCode = 0x06 // 下(04) + 左(02) = 06
	case PTZDownRight:
		cmdCode = 0x05 // 下(04) + 右(01) = 05
	case PTZZoomIn:
		cmdCode = 0x10 // Bit4: 放大
	case PTZZoomOut:
		cmdCode = 0x20 // Bit5: 缩小
	case PTZFocusNear:
		cmdCode = 0x40 // Bit6: 近焦
	case PTZFocusFar:
		cmdCode = 0x80 // Bit7: 远焦
	default:
		cmdCode = 0x00
	}

	// 限制速度范围 (0-255)
	if horizontalSpeed < 0 {
		horizontalSpeed = 0
	}
	if horizontalSpeed > 255 {
		horizontalSpeed = 255
	}
	if verticalSpeed < 0 {
		verticalSpeed = 0
	}
	if verticalSpeed > 255 {
		verticalSpeed = 255
	}

	// Zoom 速度：根据方向设置
	zoomSpeed := byte(0x10) // 默认值
	if direction == PTZStop {
		zoomSpeed = 0x00
	} else if direction == PTZZoomIn || direction == PTZZoomOut {
		zoomSpeed = byte(horizontalSpeed) // Zoom操作使用水平速度
	}

	// 构建 8 字节命令
	cmd := []byte{
		0xA5,                  // 同步头1
		0x0F,                  // 同步头2
		0x01,                  // 组合码 (云台控制)
		cmdCode,               // 命令码
		byte(horizontalSpeed), // 水平速度
		byte(verticalSpeed),   // 垂直速度
		zoomSpeed,             // Zoom/预留
		0x00,                  // 校验和 (待计算)
	}

	// 计算校验和: 前7字节累加后 mod 256 (根据WVP日志验证)
	var checksum int
	for i := 0; i < 7; i++ {
		checksum += int(cmd[i])
	}
	cmd[7] = byte(checksum % 256)

	// 返回 16 字符的十六进制字符串
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
		cmd[0], cmd[1], cmd[2], cmd[3], cmd[4], cmd[5], cmd[6], cmd[7])
}
