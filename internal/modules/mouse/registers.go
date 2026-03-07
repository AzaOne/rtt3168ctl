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

	Rate125  uint8 = 194
	Rate250  uint8 = 130
	Rate500  uint8 = 66
	Rate1000 uint8 = 2

	RGBAlwaysOn  uint8 = 0x01
	RGBBreathing uint8 = 0x41
	RGBCycle6    uint8 = 0x61
	RGBCycle12   uint8 = 0x81
	RGBCycle768  uint8 = 0xA1
	RGBOff       uint8 = 0xE1
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
