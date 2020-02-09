package main

import "time"

type Replay struct {
	Duration      time.Duration
	Version       int
	BuildNumber   int
	Expac         Expac
	IsMultiplayer bool
	isReforged 	  bool
	GameType      GameType
	PrivateGame   bool
	GameOptions   GameOptions
	Players       []Player
	Slots				  []Slot
	RandomSeed    uint32
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
	ObsOff ObserverSetting = 0
	ObsOnDefeat = 2
	ObsOn = 3
	ObsReferees = 4
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
	Name string
	SlotId int
	BattleNet *BattleNet2Account
}
func (p *Player) GetSlot(replay *Replay) *Slot {
	return &replay.Slots[p.SlotId]
}

type BattleNet2Account struct {
	PlayerId  int
	Avatar    string
	Username  string
	Clan      string
	ExtraData []byte
}

type Slot struct {
	PlayerId              int
	IsCPU                 bool
	Race                  Race
	raceSelectableOrFixed bool
	SlotStatus            SlotStatus
	TeamNumber            int
	Color                 Color
	AIStrength            AIStrength
	Handicap              int
	MapDownloadPercent    byte
}

type Race string
const (
	Human Race = "Human"
	Orc = "Orc"
	NightElf = "Night Elf"
	Undead = "Undead"
	RandomRace = "Random"
)
var races = map[byte]Race {
	0x01: Human,
	0x02: Orc,
	0x04: NightElf,
	0x08: Undead,
	0x20: RandomRace,
}


type SlotStatus byte
const (
	EmptySlot SlotStatus = 0x0
	ClosedSlot = 0x1
	UsedSlot = 0x2
)

type Color byte
const (
	Red Color = iota
	Blue
	Cyan
	Purple
	Yellow
	Orange
	Green
	Pink
	Gray
	LightBlue
	DarkGreen
	Brown
	ObserverOrRefColor
)

type AIStrength byte
const (
	EasyAI AIStrength = 0x00
	NormalAI = 0x01
	InsaneAI = 0x02
)