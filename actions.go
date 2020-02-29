package warcrumb

import (
	"bytes"
	"fmt"
	"time"
)

type Action struct {
	actionable
	Time   time.Duration
	Player *Player
}

func (a Action) MarshalText() (text []byte, err error) {
	return []byte(a.String()), nil
}

/*
func (a Action) MarshalJSON() ([]byte, error) {
	// FIXME: temporary until data is more structured
	// i.e. by default the action string representation won't be encoded
	return json.Marshal(struct {
		Time time.Duration
		Player *Player
		Data interface{}
	}{a.Time, a.Player, a.actionable})
}
*/
func (a Action) String() string {
	return fmt.Sprintf("[%s] %s: %s", a.Time, a.Player, a.actionable)
}

type actionable interface {
	fmt.Stringer
	APMChange() int
}

type Ability struct {
	AbilityFlags uint16
	ItemId       [4]byte
}

func (a Ability) String() string {
	var itemIdStr string
	if bytes.Contains(a.ItemId[:], []byte{0x00}) {
		// alphanumeric id
		itemIdStr = fmt.Sprintf("%#02v", a.ItemId)
	} else {
		// string
		var reversed [4]byte
		for i := 0; i < 4; i++ {
			reversed[i] = a.ItemId[3-i]
		}
		itemIdStr = string(reversed[:])
	}
	return fmt.Sprintf("ability (%x, %s)", a.AbilityFlags, itemIdStr)
}

func (a Ability) APMChange() int {
	return 1
}

type TargetedAbility struct {
	Ability
	X, Y int
}

func (a TargetedAbility) String() string {
	return fmt.Sprintf("targeted (%d,%d) %s", a.X, a.Y, a.Ability.String())
}

type ObjectTargetedAbility struct {
	TargetedAbility
	objectId1, objectId2 int
}

func (a ObjectTargetedAbility) String() string {
	return fmt.Sprintf("object (%d,%d) %s", a.objectId1, a.objectId2, a.TargetedAbility.String())
}

func readActionBlock(buffer *bytes.Buffer, replay *Replay) (actionable, error) {
	actionId, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	switch actionId {
	case 0x10, 0x11, 0x12, 0x13: // ability (+target) (+object target) (+ target item)
		var abilityFlags uint16
		if replay.Version < 13 {
			abilityFlagsB, err := buffer.ReadByte()
			if err != nil {
				return nil, err
			}
			abilityFlags = uint16(abilityFlagsB)
		} else {
			abilityFlags, err = readWORD(buffer)
			if err != nil {
				return nil, err
			}
		}
		var itemId [4]byte
		_, err := buffer.Read(itemId[:])
		if err != nil {
			return nil, err
		}
		//itemIdRaw, err := readDWORD(buffer); if err != nil {return nil, err}
		//binary.BigEndian.PutUint32(itemId[:], itemIdRaw)
		if replay.Version >= 7 {
			// two unknown values that are 0xFFFFFFFF in old replays (before like, 1.18)
			if _, err = readDWORD(buffer); err != nil {
				return nil, err
			}
			if _, err = readDWORD(buffer); err != nil {
				return nil, err
			}
		}
		baseAbility := Ability{AbilityFlags: abilityFlags, ItemId: itemId}
		if actionId == 0x10 {
			return baseAbility, nil
		}
		targetX, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}
		targetY, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}
		targetedAbility := TargetedAbility{Ability: baseAbility, X: int(targetX), Y: int(targetY)}
		if actionId == 0x11 {
			return targetedAbility, nil
		}

		targetObjId1, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}
		targetObjId2, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}

		if actionId == 0x12 {
			return ObjectTargetedAbility{
				TargetedAbility: targetedAbility,
				objectId1:       int(targetObjId1),
				objectId2:       int(targetObjId2),
			}, nil
		} else {
			//fmt.Println("******", actionId)
		}

	}
	return nil, nil
}
