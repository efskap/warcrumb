//	Warcrumb - Replay parser library for Warcraft 3
//	Copyright (C) 2020 Dmitry Narkevich
//
//	This program is free software: you can redistribute it and/or modify
//	it under the terms of the GNU General Public License as published by
//	the Free Software Foundation, either version 3 of the License, or
//	(at your option) any later version.
//
//	This program is distributed in the hope that it will be useful,
//	but WITHOUT ANY WARRANTY; without even the implied warranty of
//	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//	GNU General Public License for more details.
//
//	You should have received a copy of the GNU General Public License
//	along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

type GameSpeed int

type Expac int

type Visibility int

type GameType uint16

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
	Author      Slot
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

type MsgToPlayer struct{ Target Slot }

func (MsgToPlayer) isMsgDest()       {}
func (m MsgToPlayer) String() string { return "To " + m.Target.String() }

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
func (s Slot) String() string {
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

func (r Race) String() string {
	return r.name
}
func (r Race) ShortName() string {
	return string(r.name[0])
}
func (r Race) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type slotStatus string

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
