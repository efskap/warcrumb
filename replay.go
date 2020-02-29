package warcrumb

import (
	"fmt"
	"image/color"
	"time"
)

type Replay struct {
	parseOptions
	Duration       time.Duration
	Version        int
	BuildNumber    int
	Expac          Expac
	IsMultiplayer  bool
	isReforged     bool
	GameType       GameType
	PrivateGame    bool
	GameOptions    GameOptions
	Players        map[int]*Player
	Slots          []Slot
	RandomSeed     uint32
	selectMode     byte
	startSpotCount int
	ChatMessages   []ChatMessage
	Saver          *Player
	WinnerTeam     int // -1 represents a draw
	Actions        []Action
}

type parseOptions struct {
	debugMode bool
}

type GameOptions struct {
	MapName               string
	CreatorName           string
	TeamsTogether         bool
	LockTeams             bool
	FullSharedUnitControl bool
	RandomHero            bool
	RandomRaces           bool
	Speed                 GameSpeed
	Visibility            Visibility
	ObserverSetting       ObserverSetting
	GameName              string
}

type ObserverSetting int

const (
	ObsOff      ObserverSetting = 0
	ObsOnDefeat                 = 2
	ObsOn                       = 3
	ObsReferees                 = 4
)

type GameSpeed int

const (
	SlowSpeed = iota
	NormalSpeed
	FastSpeed
)

type Expac int

const (
	ReignOfChaos Expac = iota
	TheFrozenThrone
)

type Visibility int

const (
	Default Visibility = iota
	AlwaysVisible
	MapExplored
	HideTerrain
)

type GameType uint16

const (
	Unknown      GameType = 0x00
	FFAor1on1             = 0x01
	Custom                = 0x09
	Singleplayer          = 0x1D
	LadderTeam            = 0x20 // (AT or RT, 2on2/3on3/4on4)
)

type Player struct {
	Name   string
	Id     int
	SlotId int
	// slot is currently not exposed to avoid panic (due to pointer cycle) when converting to json
	// but that's probably not the best reason
	slot      *Slot
	BattleNet *BattleNet2Account
}

func (p Player) String() string {
	if p.BattleNet != nil {
		return p.BattleNet.Username
	} else {
		return p.Name
	}
}

type BattleNet2Account struct {
	PlayerId  int
	Avatar    string
	Username  string
	Clan      string
	ExtraData []byte
}

type ChatMessage struct {
	Timestamp   time.Duration
	Author      *Player
	Body        string
	Destination MsgDestination
}
type MsgDestination interface {
	isMsgDest() // dummy method to emulate sum type
	fmt.Stringer
}
type MsgToEveryone struct{}

func (MsgToEveryone) isMsgDest()     {}
func (MsgToEveryone) String() string { return "All" }

type MsgToAllies struct{}

func (MsgToAllies) isMsgDest()     {}
func (MsgToAllies) String() string { return "Allies" }

type MsgToObservers struct{}

func (MsgToObservers) isMsgDest()     {}
func (MsgToObservers) String() string { return "Observers" }

type MsgToPlayer struct{ *Player }

func (MsgToPlayer) isMsgDest()       {}
func (m MsgToPlayer) String() string { return "To " + m.Player.Name }

type Slot struct {
	Id                    int
	Player                *Player
	IsCPU                 bool
	Race                  Race
	raceSelectableOrFixed bool
	SlotStatus            slotStatus
	TeamNumber            int
	Color                 Color
	AIStrength            AIStrength
	Handicap              int
	MapDownloadPercent    byte
	playerId              int
}

// String returns the text you'd see in-game as the name of that slot.
func (s *Slot) String() string {
	switch s.SlotStatus {
	case EmptySlot:
		return "Open"
	case ClosedSlot:
		return "Closed"
	case UsedSlot:
		if s.IsCPU {
			return fmt.Sprintf("Computer (%s)", s.AIStrength.String())
		} else {
			return s.Player.String()
		}
	}
	return ""
}

type Race struct {
	name string
	// could add an icon field
}

var (
	Human      = Race{"Human"}
	Orc        = Race{"Orc"}
	NightElf   = Race{"Night Elf"}
	Undead     = Race{"Undead"}
	RandomRace = Race{"Random"}
)

func (r Race) String() string {
	return r.name
}
func (r Race) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

var races = map[byte]Race{
	0x01: Human,
	0x02: Orc,
	0x04: NightElf,
	0x08: Undead,
	0x20: RandomRace,
}

type slotStatus string

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

type Color struct {
	color.Color
	name string
}

func (c Color) String() string {
	return c.name
}
func (c Color) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

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

type AIStrength byte

func (a AIStrength) String() string {
	switch a {
	case EasyAI:
		return "Easy"
	case NormalAI:
		return "Normal"
	case InsaneAI:
		return "Insane"
	}
	//return fmt.Sprintf("n/a (0x%x)", *a)
	return "n/a"
}
func (a AIStrength) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

const (
	EasyAI   AIStrength = 0x00
	NormalAI            = 0x01
	InsaneAI            = 0x02
)
