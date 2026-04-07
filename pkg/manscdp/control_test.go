package manscdp

import (
	"testing"
)

func TestBuildPTZCmd(t *testing.T) {
	tests := []struct {
		name            string
		direction       PTZDirection
		hSpeed          int
		vSpeed          int
		expectedPattern string // 预期的十六进制字符串 (WVP验证格式)
	}{
		{
			name:      "向左命令 (WVP验证)",
			direction: PTZLeft,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+02+1E+1E+10 = 259 mod 256 = 03
			expectedPattern: "A50F01021E1E1003",
		},
		{
			name:      "停止命令 (WVP验证)",
			direction: PTZStop,
			hSpeed:    0,
			vSpeed:    0,
			// A5+0F+01+00+00+00+00 = 181 = B5
			expectedPattern: "A50F0100000000B5",
		},
		{
			name:      "向上命令",
			direction: PTZUp,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+08+1E+1E+10 = 265 mod 256 = 09
			expectedPattern: "A50F01081E1E1009",
		},
		{
			name:      "向右命令",
			direction: PTZRight,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+01+1E+1E+10 = 258 mod 256 = 02
			expectedPattern: "A50F01011E1E1002",
		},
		{
			name:      "向下命令",
			direction: PTZDown,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+04+1E+1E+10 = 261 mod 256 = 05
			expectedPattern: "A50F01041E1E1005",
		},
		{
			name:      "放大命令",
			direction: PTZZoomIn,
			hSpeed:    0,
			vSpeed:    0,
			// Zoom操作: zoomSpeed = hSpeed = 0
			// A5+0F+01+10+00+00+00 = 165+15+1+16+0+0+0 = 197 = C5
			expectedPattern: "A50F0110000000C5",
		},
		{
			name:      "缩小命令",
			direction: PTZZoomOut,
			hSpeed:    0,
			vSpeed:    0,
			// Zoom操作: zoomSpeed = hSpeed = 0
			// A5+0F+01+20+00+00+00 = 165+15+1+32+0+0+0 = 213 = D5
			expectedPattern: "A50F0120000000D5",
		},
		{
			name:      "上左组合命令",
			direction: PTZUpLeft,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+0A+1E+1E+10 = 267 mod 256 = 0B
			expectedPattern: "A50F010A1E1E100B",
		},
		{
			name:      "上右组合命令",
			direction: PTZUpRight,
			hSpeed:    30,
			vSpeed:    30,
			// A5+0F+01+09+1E+1E+10 = 266 mod 256 = 0A
			expectedPattern: "A50F01091E1E100A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPTZCmd(tt.direction, tt.hSpeed, tt.vSpeed)
			if result != tt.expectedPattern {
				t.Errorf("BuildPTZCmd(%v, %d, %d) = %s, want %s",
					tt.direction, tt.hSpeed, tt.vSpeed, result, tt.expectedPattern)
			}
		})
	}
}

func TestBuildPTZCmdChecksum(t *testing.T) {
	// 验证校验和计算是否正确 (累加mod256)
	// 向左: A5+0F+01+02+1E+1E+10 = 165+15+1+2+30+30+16 = 259 mod 256 = 03
	leftCmd := BuildPTZCmd(PTZLeft, 30, 30)
	if leftCmd != "A50F01021E1E1003" {
		t.Errorf("向左命令格式错误: %s", leftCmd)
	}

	// 停止: A5+0F+01+00+00+00+00 = 165+15+1 = 181 = B5
	stopCmd := BuildPTZCmd(PTZStop, 0, 0)
	if stopCmd != "A50F0100000000B5" {
		t.Errorf("停止命令格式错误: %s", stopCmd)
	}
}

func TestBuildPTZCmdSpeedLimit(t *testing.T) {
	// 测试速度边界
	cmd1 := BuildPTZCmd(PTZUp, -1, -1)
	// 速度被限制为 0: zoomSpeed=0x10(非Zoom操作)
	// A5+0F+01+08+00+00+10 = 165+15+1+8+0+0+16 = 205 = CD
	if cmd1 != "A50F0108000010CD" {
		t.Errorf("负速度应被限制为0: %s", cmd1)
	}

	cmd2 := BuildPTZCmd(PTZUp, 300, 300)
	// 速度被限制为 255 (FF): zoomSpeed=0x10(非Zoom操作)
	// A5+0F+01+08+FF+FF+10 = 165+15+1+8+255+255+16 = 715 mod 256 = 203 = CB
	if cmd2 != "A50F0108FFFF10CB" {
		t.Errorf("超限速度应被限制为255: %s", cmd2)
	}
}
