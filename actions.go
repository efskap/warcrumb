package warcrumb

import (
	"bytes"
	"fmt"
	"strconv"
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
type ItemId [4]byte

func (a ItemId) String() string {
	//if bytes.Equal(a[:], []byte{0x03, 0x00, 0x0d, 0x00}) {
	//	return "right clicked"
	//}
	if full, ok := byteStrings[string(a[:])]; ok {
		return full.Tip
	}
	if bytes.Contains(a[:], []byte{0x00}) {
		// alphanumeric id
		return string(a[:])
		//return fmt.Sprintf("%#02v", a)
	} else {
		// string
		var reversed [4]byte
		for i := 0; i < 4; i++ {
			reversed[i] = a[3-i]
		}
		str := string(reversed[:])
		if full, ok := WC3Strings[str]; ok {
			return full.Tip
		}
		return str
	}
}

type ObjectId uint32

func (o ObjectId) IsGround() bool {
	return o == 0xFFFFFFFF
}
func (o ObjectId) String() string {
	if o.IsGround() {
		return "the ground"
	} else {
		return strconv.Itoa(int(o))
	}
}

type Ability struct {
	AbilityFlags uint16
	ItemId       ItemId
}

func (a Ability) String() string {
	return fmt.Sprintf("Ability [mod %#02x] %s", a.AbilityFlags, a.ItemId)
}

func (a Ability) APMChange() int {
	return 1
}

type TargetedAbility struct {
	Ability
	Target PointF
}

func (a TargetedAbility) String() string {
	return fmt.Sprintf("%s at %s", a.Ability, a.Target)
}

type ObjectTargetedAbility struct {
	TargetedAbility
	TargetObjectId1, TargetObjectId2 ObjectId
}

func (a ObjectTargetedAbility) String() string {
	return fmt.Sprintf("%s object (%s, %s) at %s", a.Ability, a.TargetObjectId1, a.TargetObjectId2, a.Target)
}

func (a ObjectTargetedAbility) TargetsGround() bool {
	return a.TargetObjectId1.IsGround() && a.TargetObjectId2.IsGround()
}

type GiveOrDropItem struct {
	ObjectTargetedAbility
	ItemObjectId1, ItemObjectId2 ObjectId
}

func (g GiveOrDropItem) String() string {
	if g.TargetsGround() {
		return fmt.Sprintf("Drop item %s (%s, %s) on ground at %s", g.ItemId, g.ItemObjectId1, g.ItemObjectId2, g.Target)
	}
	return fmt.Sprintf("Give item %s (%s, %s) to obj (%s, %s) at %s", g.ItemId, g.ItemObjectId1, g.ItemObjectId2, g.TargetObjectId1, g.TargetObjectId2, g.Target)
}

type TwoTargetTwoItemAblity struct {
	TargetedAbility
	ItemId2 ItemId
	Target2 PointF
}

func (t TwoTargetTwoItemAblity) String() string {
	return fmt.Sprintf("%s + %s to %s & %s", t.Ability, t.ItemId2, t.Target, t.Target2)
}

func readActionBlock(buffer *bytes.Buffer, replay *Replay) (actionable, error) {
	actionId, err := buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	switch actionId {
	case 0x10, 0x11, 0x12, 0x13, 0x14: // ability (+target) (+object target) (+ target item)
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
		if _, err = buffer.Read(itemId[:]); err != nil {
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
		target, err := readPointF(buffer)
		if err != nil {
			return nil, err
		}

		targetedAbility := TargetedAbility{Ability: baseAbility, Target: target}
		if actionId == 0x11 {
			return targetedAbility, nil
		}

		if actionId == 0x14 {
			var itemId2 [4]byte
			if _, err = buffer.Read(itemId2[:]); err != nil {
				return nil, err
			}
			buffer.Next(9)
			target2, err := readPointF(buffer)
			if err != nil {
				return nil, err
			}

			return TwoTargetTwoItemAblity{
				TargetedAbility: targetedAbility,
				ItemId2:         itemId2,
				Target2:         target2,
			}, nil

		}

		targetObjId1, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}
		targetObjId2, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}

		objTargetedAbility := ObjectTargetedAbility{
			TargetedAbility: targetedAbility,
			TargetObjectId1: ObjectId(targetObjId1),
			TargetObjectId2: ObjectId(targetObjId2),
		}
		if actionId == 0x12 {
			return objTargetedAbility, nil
		}

		itemObjId1, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}
		itemObjId2, err := readDWORD(buffer)
		if err != nil {
			return nil, err
		}

		return GiveOrDropItem{objTargetedAbility, ObjectId(itemObjId1), ObjectId(itemObjId2)}, nil

	}
	return nil, nil
}

// StringsEntity represents a definition from the WC3 *strings.txt files, which tools/gen_strings uses to generate mappings.
// Replay files only contain short codes, e.g. "hpea", from which we want to get "Peasant" (and possibly other data later).
type StringsEntity struct {
	// Name is the full name of the entity, e.g. "Peasant"
	Name string
	// Code is the 4-char string it is stored under, e.g. "hpea"
	Code string
	// Tip is the imperative form, e.g. "Train Peasant"
	Tip string
}

var byteStrings = map[string]StringsEntity{
	"\u0003\u0000\r\u0000": {
		Name: "Right Click",
		Code: "",
		Tip:  "Right-click",
	},
	"\u000f\u0000\r\u0000": {
		Name: "Attack",
		Tip:  "Attack",
	},
}
