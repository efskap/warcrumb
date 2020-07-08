package warcrumb

import "image/color"

// colour values from WorldEdit, names from https://gaming-tools.com/warcraft-3/patch-1-29/
var (
	Red       = Color{color.RGBA{255, 4, 2, 255}, "Red"}
	Blue      = Color{color.RGBA{0, 66, 255, 255}, "Blue"}
	Teal      = Color{color.RGBA{27, 230, 186, 255}, "Teal"}
	Purple    = Color{color.RGBA{85, 0, 129, 255}, "Purple"}
	Yellow    = Color{color.RGBA{255, 252, 0, 255}, "Yellow"}
	Orange    = Color{color.RGBA{255, 138, 13, 255}, "Orange"}
	Green     = Color{color.RGBA{32, 191, 0, 255}, "Green"}
	Pink      = Color{color.RGBA{227, 91, 175, 255}, "Pink"}
	Grey      = Color{color.RGBA{148, 150, 151, 255}, "Grey"}
	LightBlue = Color{color.RGBA{126, 191, 241, 255}, "LightBlue"}
	DarkGreen = Color{color.RGBA{16, 98, 71, 255}, "DarkGreen"}
	Brown     = Color{color.RGBA{79, 43, 5, 255}, "Brown"}
	Maroon    = Color{color.RGBA{156, 0, 0, 255}, "Maroon"}
	Navy      = Color{color.RGBA{0, 0, 194, 255}, "Navy"}
	Turquoise = Color{color.RGBA{0, 235, 255, 255}, "Turquoise"}
	Violet    = Color{color.RGBA{189, 0, 255, 255}, "Violet"}
	Wheat     = Color{color.RGBA{236, 204, 134, 255}, "Wheat"}
	Peach     = Color{color.RGBA{247, 164, 139, 255}, "Peach"}
	Mint      = Color{color.RGBA{191, 255, 128, 255}, "Mint"}
	Lavender  = Color{color.RGBA{219, 184, 236, 255}, "Lavender"}
	Coal      = Color{color.RGBA{79, 79, 85, 255}, "Coal"}
	Snow      = Color{color.RGBA{236, 240, 255, 255}, "Snow"}
	Emerald   = Color{color.RGBA{0, 120, 30, 255}, "Emerald"}
	Peanut    = Color{color.RGBA{164, 111, 52, 255}, "Peanut"}
)

var (
	Human      = Race{"Human"}
	Orc        = Race{"Orc"}
	NightElf   = Race{"Night Elf"}
	Undead     = Race{"Undead"}
	RandomRace = Race{"Random"}
)

var races = map[byte]Race{
	0x01: Human,
	0x02: Orc,
	0x04: NightElf,
	0x08: Undead,
	0x20: RandomRace,
}

var slotStatuses = map[byte]slotStatus{
	0x0: EmptySlot,
	0x1: ClosedSlot,
	0x2: UsedSlot,
}

const (
	EmptySlot  slotStatus = "Empty"
	ClosedSlot            = "Closed"
	UsedSlot              = "Used"
)

// colors stores all colors in order for lookup (e.g. Red is encoded as 0x00, Blue as 0x01)
var colors = []Color{
	Red,
	Blue,
	Teal,
	Purple,
	Yellow,
	Orange,
	Green,
	Pink,
	Grey,
	LightBlue,
	DarkGreen,
	Brown,
	Maroon, // Observer or Ref color btw
	Navy,
	Turquoise,
	Violet,
	Wheat,
	Peach,
	Mint,
	Lavender,
	Coal,
	Snow,
	Emerald,
	Peanut,
}

const (
	EasyAI   AIStrength = 0x00
	NormalAI            = 0x01
	InsaneAI            = 0x02
)

const (
	ObsOff      ObserverSetting = 0
	ObsOnDefeat                 = 2
	ObsOn                       = 3
	ObsReferees                 = 4
)

const (
	SlowSpeed = iota
	NormalSpeed
	FastSpeed
)

const (
	ReignOfChaos Expac = iota
	TheFrozenThrone
)

const (
	Default Visibility = iota
	AlwaysVisible
	MapExplored
	HideTerrain
)

const (
	Unknown      GameType = 0x00
	FFAor1on1             = 0x01
	Custom                = 0x09
	Singleplayer          = 0x1D
	LadderTeam            = 0x20 // (AT or RT, 2on2/3on3/4on4)
)
