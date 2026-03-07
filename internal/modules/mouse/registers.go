package mouse

const (
	ReqTypeWrite = 0x40 // Vendor | Device | Out
	ReqTypeRead  = 0xC0 // Vendor | Device | In

	ReqCodeControl uint8 = 1
	ReqCodeReset   uint8 = 6

	ControlValDefault uint16 = 0x0100
	ControlIdxBank0   uint16 = 127   // 0x007F
	ControlIdxBank1   uint16 = 383   // 0x017F
	ControlIdxBank1IO uint16 = 8201  // 0x2009
	ControlIdxUnlock  uint16 = 23049 // 0x5A09
	ControlIdxLock    uint16 = 9     // 0x0009

	RegRGBSpeed  uint16 = 1
	RegDPI1      uint16 = 2
	RegDPI2      uint16 = 3
	RegDPI3      uint16 = 4
	RegDPI4      uint16 = 5
	RegDPI5      uint16 = 6 // buggy, in oficial software not used
	RegDPI6      uint16 = 7 // buggy, in oficial software not used
	RegDPISelect uint16 = 9
	RegRGBMode   uint16 = 10
	RegCPIButton uint16 = 11
	RegRate      uint16 = 14
	RegSensorID  uint16 = 29

	// Experimental/inferred runtime registers (read-only use).
	RegExpB1ButtonsMask       uint16 = 40  // 0x28
	RegExpB1ButtonsStateA     uint16 = 42  // 0x2A
	RegExpB1ButtonsStateB     uint16 = 43  // 0x2B
	RegExpB1EventState        uint16 = 117 // 0x75
	RegExpB1ButtonsMaskMirror uint16 = 168 // 0xA8
	RegExpB1ButtonsStateAMirr uint16 = 170 // 0xAA
	RegExpB1ButtonsStateBMirr uint16 = 171 // 0xAB
	RegExpB1EventStateMirror  uint16 = 245 // 0xF5

	RegExpB0MoveX            uint16 = 3   // 0x03
	RegExpB0MoveY            uint16 = 4   // 0x04
	RegExpB0EventLatch       uint16 = 8   // 0x08
	RegExpB0MoveXMirror      uint16 = 19  // 0x13
	RegExpB0EventGroup       uint16 = 51  // 0x33
	RegExpB0EventStateC      uint16 = 97  // 0x61
	RegExpB0EventStateA      uint16 = 107 // 0x6B
	RegExpB0EventStateB      uint16 = 108 // 0x6C
	RegExpB0EventLatchMirror uint16 = 136 // 0x88
	RegExpB0MoveYMirror      uint16 = 147 // 0x93
	RegExpB0EventGroupMirror uint16 = 179 // 0xB3
	RegExpB0EventStateCMirr  uint16 = 225 // 0xE1
	RegExpB0EventStateAMirr  uint16 = 235 // 0xEB
	RegExpB0EventStateBMirr  uint16 = 236 // 0xEC

	Rate125  uint8 = 194
	Rate250  uint8 = 130
	Rate500  uint8 = 66
	Rate1000 uint8 = 2

	RGBAlwaysOn      uint8 = 0x01
	RGBBreath        uint8 = 0x21
	RGBBreathSegment uint8 = 0x41
	RGBCycle6        uint8 = 0x61
	RGBCycle12       uint8 = 0x81
	RGBCycle768      uint8 = 0xA1
	RGBOff           uint8 = 0xE1
)

var CPIActionMap = map[string]uint8{
	"backward":     224,
	"forward":      225,
	"ctrl":         226,
	"win":          227,
	"browser":      228,
	"double_click": 229,
	"sniper":       230,
	"rgb_switch":   231,
	"dpi_cycle":    232,
	"play_pause":   236,
	"mute":         237,
	"next_track":   238,
	"prev_track":   239,
	"stop":         240,
	"vol_up":       242,
	"vol_down":     243,
	"win_d":        245,
	"copy":         246,
	"paste":        247,
	"prev_page":    248,
	"next_page":    249,
	"my_computer":  250,
	"calculator":   251,
	"ctrl_w":       252,
}
