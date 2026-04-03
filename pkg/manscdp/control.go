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
	XMLName  xml.Name `xml:"Control"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	PTZCmd   *PTZCmd  `xml:"PTZCmd,omitempty"`
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
	}
}

// BuildPTZCmd 构建云台控制命令字符串
// GB28181 云台控制命令格式: AABBCDDD
// AA: 方向 (00=停止, 01=上, 02=下, 04=左, 08=右, 组合方向为方向值相加)
// BB: 预置点编号 (00-FF)
// C: 水平速度 (0-F)
// DDD: 垂直速度 (000-255)
func BuildPTZCmd(direction PTZDirection, horizontalSpeed, verticalSpeed int) string {
	var directionCode int
	switch direction {
	case PTZStop:
		directionCode = 0
	case PTZUp:
		directionCode = 1
	case PTZDown:
		directionCode = 2
	case PTZLeft:
		directionCode = 4
	case PTZRight:
		directionCode = 8
	case PTZUpLeft:
		directionCode = 5 // 1 + 4
	case PTZUpRight:
		directionCode = 9 // 1 + 8
	case PTZDownLeft:
		directionCode = 6 // 2 + 4
	case PTZDownRight:
		directionCode = 10 // 2 + 8
	case PTZZoomIn:
		directionCode = 16
	case PTZZoomOut:
		directionCode = 32
	case PTZFocusNear:
		directionCode = 64
	case PTZFocusFar:
		directionCode = 128
	default:
		directionCode = 0
	}

	// 限制速度范围
	if horizontalSpeed < 0 {
		horizontalSpeed = 0
	}
	if horizontalSpeed > 15 {
		horizontalSpeed = 15
	}
	if verticalSpeed < 0 {
		verticalSpeed = 0
	}
	if verticalSpeed > 255 {
		verticalSpeed = 255
	}

	// 构建命令字符串: AABBCDDD
	// AA: 方向 (2位十六进制)
	// BB: 预置点 (固定00)
	// C: 水平速度 (1位十六进制)
	// DDD: 垂直速度 (3位十进制)
	return fmt.Sprintf("%02X00%1X%03d", directionCode, horizontalSpeed, verticalSpeed)
}
