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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/bits"
	"os"
	"strings"
	"time"
)

// ParseReplayDebug is the same as ParseReplay but dumps binaries and prints to stdout too
func ParseReplayDebug(file io.Reader) (rep Replay, err error) {
	rep.parseOptions.debugMode = true
	err = read(file, &rep)
	return rep, err
}

// ParseReplay parses an opened .w3g file.
func ParseReplay(file io.Reader) (rep Replay, err error) {
	err = read(file, &rep)
	return rep, err
}

func read(file io.Reader, rep *Replay) (err error) {
	header, err := readHeader(file)
	if err != nil {
		return fmt.Errorf("error reading header: %w", err)
	}
	rep.IsMultiplayer = header.IsMultiplayer
	rep.Duration = header.Duration
	rep.Version = header.GameVersion
	rep.Expac = header.Expac
	rep.BuildNumber = header.BuildNumber

	rep.isReforged = rep.Version >= 10032

	// might as well allocate the right size buffer based on the assumption that every block is 8K
	buffer := bytes.NewBuffer(make([]byte, 0, header.NumberOfBlocks*0x2000))
	for i := 0; i < int(header.NumberOfBlocks); i++ {
		b, err := readCompressedBlock(file, rep.isReforged)
		if err != nil {
			return fmt.Errorf("failed to decompress block i=%d: %w", i, err)
		}
		buffer.Write(b)
	}
	bufferCopy := make([]byte, buffer.Len())
	copy(bufferCopy, buffer.Bytes())

	bufferLen := buffer.Len()
	err = readDecompressedData(buffer, rep)
	if rep.debugMode {
		_ = os.Mkdir("hexdumps", os.ModePerm)
		_ = ioutil.WriteFile(fmt.Sprintf("./hexdumps/decompresssed_%s_%s.hex", rep.GameOptions.GameName, rep.GameOptions.CreatorName), bufferCopy, os.ModePerm)
	}
	if err != nil {
		readBytes := bufferLen - buffer.Len()
		return fmt.Errorf("error in decompressed data at/before %#x: %w", readBytes, err)
	}

	return
}

func readHeader(file io.Reader) (header header, err error) {
	magicString := make([]byte, 28)
	if _, err = file.Read(magicString); err != nil {
		return header, fmt.Errorf("error reading magic string: %w", err)
	}

	expected := []byte("Warcraft III recorded game\x1A\x00")
	if !bytes.Equal(magicString, expected) {
		return header, fmt.Errorf("does not seem to be a WC3 replay (incorrect magic string at start)")
	}

	headerSize, err := readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading header size: %w", err)
	}

	if headerSize != 0x40 && headerSize != 0x44 {
		fmt.Printf("Warning: unexpected header size: 0x%x\n", headerSize)
	}
	header.Length = headerSize

	_, err = readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading compressed file size: %w", err)
	}

	replayHeaderVersion, err := readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading compressed file size: %w", err)
	}
	if replayHeaderVersion > 0x01 {
		fmt.Printf("Warning: unexpected replay header version: 0x%x\n", headerSize)
	}

	_, err = readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading decompressed data size: %w", err)
	}
	nBlocks, err := readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading number of compressed blocks: %w", err)
	}
	header.NumberOfBlocks = nBlocks

	// subheader
	header.HeaderVersion = replayHeaderVersion
	if replayHeaderVersion == 0x0 {
		// This header was used for all replays saved with WarCraft III patch version v1.06 and below.
		_, err = readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading unknown field: %w", err)
		}

		version, err := readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading version number: %w", err)
		}
		header.GameVersion = int(version)
		buildNum, err := readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading build number: %w", err)
		}
		header.BuildNumber = int(buildNum)
		flags, err := readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading flags: %w", err)
		}

		header.IsMultiplayer = flags == 0x8000

		lenMS, err := readDWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading replay duration: %w", err)
		}
		header.Duration = time.Millisecond * time.Duration(lenMS)

		_, err = readDWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading checksum: %w", err)
		}
	} else if replayHeaderVersion == 0x1 {
		versionId, err := readLittleEndianString(file, 4)
		if err != nil {
			return header, fmt.Errorf("error reading version identifier: %w", err)
		}
		if versionId == "WAR3" {
			header.Expac = ReignOfChaos
		} else if versionId == "W3XP" {
			header.Expac = TheFrozenThrone
		}

		version, err := readDWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading version number: %w", err)
		}
		header.GameVersion = int(version)

		buildNum, err := readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading build number: %w", err)
		}
		header.BuildNumber = int(buildNum)

		flags, err := readWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading flags: %w", err)
		}

		header.IsMultiplayer = flags == 0x8000

		lenMS, err := readDWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading replay duration: %w", err)
		}
		header.Duration = time.Millisecond * time.Duration(lenMS)

		_, err = readDWORD(file)
		if err != nil {
			return header, fmt.Errorf("error reading checksum: %w", err)
		}

	} else {
		return header, fmt.Errorf("unsupported header version: 0x%x", replayHeaderVersion)
	}
	return header, nil
}

func readDecompressedData(buffer *bytes.Buffer, rep *Replay) error {

	_, err := readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading unknown field: %w", err)
	}
	playerRecords := make(map[int]*playerRecord)
	// [playerRecord]
	if err = expectByte(buffer, 0); err != nil {
		return err
	}
	p, err := readPlayerRecord(buffer, rep)
	if err != nil {
		return err
	}
	playerRecords[p.Id] = &p

	gameName, err := buffer.ReadString(0) // read null terminated string
	if err != nil {
		return fmt.Errorf("error reading game name: %w", err)
	}
	rep.GameOptions.GameName = strings.TrimRight(gameName, "\000")

	// skip null byte normally, but this can also be... "hunter2". srsly
	if b, err := buffer.ReadByte(); err != nil {
		return err
	} else if b != 0 {
		str, err := buffer.ReadString(0)
		if err != nil {
			return fmt.Errorf("error reading mystery string: %w", err)
		}

		// add the byte we removed back to the beginning
		if rep.debugMode {
			str = strings.TrimRight(string(append([]byte{b}, str...)), "\000")
			fmt.Println("mystery string:", str)
		}
	}

	encodedString, err := buffer.ReadString(0) // read null terminated string
	if err != nil {
		return fmt.Errorf("error reading encoded string: %w", err)
	}

	if err = readEncodedString(encodedString, rep); err != nil {
		return fmt.Errorf("error reading decoded string: %w", err)
	}

	// [PlayerCount]
	_, err = readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading player count: %w", err)
	}
	//rep.Slots = make([]Slot, playerCount)

	//fmt.Println("playercount", playerCount)
	// [GameType]

	gameType, err := buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading game type: %w", err)
	}
	rep.GameType = GameType(gameType)

	privateFlag, err := buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading private flag: %w", err)
	}
	//fmt.Printf("private flag: 0x%x\n", privateFlag)
	rep.PrivateGame = privateFlag == 0x08 || privateFlag == 0xc8
	// TODO: this can also be 0x20 (in reforged public custom game) or 0x40 (reforged matchmaking)

	if err = expectWORD(buffer, 0); err != nil {
		var unexpectedValueError UnexpectedValueError
		if errors.As(err, &unexpectedValueError) {
			//fmt.Printf("Unknown byte in 4.7 [GameType] is not 0 but 0x%x!\n", unexpectedValueError.actual)
		} else {
			//return err
		}
	}

	// this is called LanguageID in the txt file but don't think there's a use for it
	_, err = readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading LanguageID: %w", err)
	}
	//fmt.Printf("LanguageID (?) = 0x%x\n", unknownMaybeLangId)

	// player record, bnet, break on gamestartrecord
	for {
		recordId, err := buffer.ReadByte()
		if err != nil {
			return fmt.Errorf("error reading record id: %w", err)
		}

		if recordId == 0x16 {
			// playerRecord
			if pRec, err := readPlayerRecord(buffer, rep); err != nil {
				return err
			} else {
				playerRecords[pRec.Id] = &pRec
			}
			if _, err = readDWORD(buffer); err != nil {
				return err
			}
		} else if recordId == 0x39 {
			if rep.debugMode {
				fmt.Println("[*] Battle.net 2.0 data present")
			}
			// TODO: give this var a more... semantic name
			after39, err := buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading value bnet section kind: %w", err)
			}
			if after39 == 4 || after39 == 5 {
				// some sort of bonus data that needs further investigation

				// now, online games seem to have this at 0 while LAN ones have 2 sometimes
				bonusDataLength, err := readDWORD(buffer)
				if err != nil {
					return fmt.Errorf("error reading bnet bonus data length: %w", err)
				}
				// not sure how to use the following data so skip it for now
				bonusData := make([]byte, bonusDataLength)
				_, err = buffer.Read(bonusData)
				if err != nil {
					return fmt.Errorf("error reading bonus data: %w", err)
				}
			} else if after39 == 3 {
				lengthOfBnetBlock, err := readDWORD(buffer)
				if err != nil {
					return fmt.Errorf("error reading bnet2.0 block length: %w", err)
				}
				// and indeed if we just read the rest of the bnet block, we go straight to GameStartRecord
				bnetBlock := make([]byte, lengthOfBnetBlock)
				_, err = buffer.Read(bnetBlock)
				if err != nil {
					return fmt.Errorf("error reading bnet2.0 block: %w", err)
				}

				if rep.debugMode {
					_ = ioutil.WriteFile("hexdumps/bnetBlock.hex", bnetBlock, os.ModePerm)
				}
				bnetBuffer := bytes.NewBuffer(bnetBlock)

				// now we don't know how many account entries are in the block
				// but we know the size of the whole thing, so just read it until it's empty
				// each iteration reads one account
				for bnetBuffer.Len() > 0 {
					var acct BattleNet2Account
					// peek ahead
					// seems like if there's an 0A, there's a sort of "unwrapping" we have to do
					// before calling the "inner" function
					// BNet games have 0x0A, but Reforged LAN ones don't.
					if bnetBuffer.Bytes()[0] == 0x0A {
						acct, err = readBattleNetAcct(bnetBuffer)
					} else {
						acct, err = readBnetAcctInner(bnetBuffer)
					}
					if err != nil {
						return fmt.Errorf("error reading bnet2.0 accounts: %w", err)
					}
					pRec, ok := playerRecords[acct.PlayerId]
					if !ok {
						return fmt.Errorf("bnet2.0 account refers to nonexistent playerRecord: %d", acct.PlayerId)
					}
					pRec.Bnet2Acc = &acct
					playerRecords[acct.PlayerId] = pRec
				}
			} else {
				return fmt.Errorf("unexpected byte after 0x39 in bnet section: %#02x", after39)
			}

		} else if recordId == 0x19 {
			break
		} else {
			fmt.Printf("Not sure how to handle recordId 0x%x\n", recordId)
			break
		}
	}

	// GameStartRecord

	if _, err := readWORD(buffer); err != nil {
		return err
	} else {
		//fmt.Println(dataBytes, "data bytes")
	}
	nr, err := buffer.ReadByte()
	if err != nil {
		return err
	} else {
		//fmt.Println(nr, "slot records")
	}
	rep.Slots = make([]Slot, 0, nr)
	for slotId := 0; slotId < int(nr); slotId++ {
		playerId, err := buffer.ReadByte()
		if err != nil {
			return err
		}

		slotRecord := Slot{Id: slotId}
		// if playerId == 0, it's a computer and thus won't be in playerRecords
		if playerId != 0 {
			pRec, ok := playerRecords[int(playerId)]
			if !ok {
				return fmt.Errorf("slot references invalid player record: id=%d", playerId)
			}
			pRec.SlotId = slotId
			slotRecord.playerId = pRec.Id
		}
		if mapDownloadPct, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.MapDownloadPercent = mapDownloadPct
			if !(mapDownloadPct == 255 || mapDownloadPct == 100) {
				return fmt.Errorf("sanity check failed: playerId = %d, map download %% = 0x%x", playerId, mapDownloadPct)
			}
		}
		if slotStatus, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotStatus, ok := slotStatuses[slotStatus]
			if !ok {
				return fmt.Errorf("invalid slot status: 0x%x", slotStatus)
			}
			slotRecord.SlotStatus = slotStatus
		}
		if isCPU, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.IsCPU = isCPU == 1
			if slotRecord.IsCPU != (playerId == 0) {
				//return fmt.Errorf("iff CPU, playerId should be 0 but it was %d", playerId)
			}
		}
		if teamNumber, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.TeamNumber = int(teamNumber) + 1 // inside warcrumb teams are 1 indexed!!!
		}
		if color, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.Color = colors[color]
		}

		if playerRace, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			if playerRace&0x40 > 0 {
				slotRecord.raceSelectableOrFixed = true
				playerRace -= 0x40
			}
			race, ok := races[playerRace]
			if !ok {
				return fmt.Errorf("unknown race: 0x%x", playerRace)
			} else {
				slotRecord.Race = race
			}
		}
		if rep.Version >= 03 {
			if aiStrength, err := buffer.ReadByte(); err != nil {
				return err
			} else {
				slotRecord.AIStrength = AIStrength(aiStrength)
				/*
					if !slotRecord.IsCPU && aiStrength != 0x01 {
							return fmt.Errorf("if not CPU, aiStrshould be 0x01 but it was 0x%x", aiStrength)
						}
				*/
			}
		}
		if rep.Version >= 07 {
			if playerHandicap, err := buffer.ReadByte(); err != nil {
				return err
			} else {
				slotRecord.Handicap = int(playerHandicap)
			}
		}
		rep.Slots = append(rep.Slots, slotRecord)
	}

	rep.Players = make(map[int]*Player)
	for id, pRec := range playerRecords {
		if id != pRec.Id || rep.Slots[pRec.SlotId].playerId != id {
			return fmt.Errorf("id was not set correctly")
		}
		rep.Players[id] = &Player{
			Id:        id,
			SlotId:    pRec.SlotId,
			BattleNet: pRec.Bnet2Acc,
			Name:      pRec.Name,
			slot:      &rep.Slots[pRec.SlotId],
		}
	}

	for i, slot := range rep.Slots {
		player, ok := rep.Players[slot.playerId]
		if ok {
			rep.Slots[i].Player = player
		}

	}

	// make sure slots and players refer to each other consistently
	// note that unoccupied slots refer to player 0 and aren't checked here
	for _, p := range rep.Players {
		if p.slot.Player.Id != p.Id {
			return fmt.Errorf("player %+v and Slot %+v ids aren't consistent", p, p.slot)
		}
	}

	if randomSeed, err := readDWORD(buffer); err != nil {
		return fmt.Errorf("error reading random seed: %w", err)
	} else {
		rep.RandomSeed = randomSeed
	}

	if selectMode, err := buffer.ReadByte(); err != nil {
		return fmt.Errorf("error reading select mode: %w", err)
	} else {
		rep.selectMode = selectMode
		// TODO
		/*
		   0x00 - team & race selectable (for standard custom games)
		   0x01 - team not selectable
		          (map setting: fixed alliances in WorldEditor)
		   0x03 - team & race not selectable
		          (map setting: fixed player properties in WorldEditor)
		   0x04 - race fixed to random
		          (extended map options: random races selected)
		   0xcc - Automated Match Making (ladder)
		*/
	}

	if startSpotCount, err := buffer.ReadByte(); err != nil {
		return fmt.Errorf("error reading start spot count: %w", err)
	} else {
		rep.startSpotCount = int(startSpotCount)
	}

	zeroes := 0
	currentTimeMS := 0
	var leaveUnknown uint32 // "unknown" variable from LeaveGame that we check for increment
	numLeaves := 0
	saverWon := false // with LeaveGame{0x0C (not last), 0x09}, we know the saver won, but we don't know who they are yet

	// ReplayData blocks
	for {
		blockId, err := buffer.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error reading block id: %w", err)
		}

		if rep.Version < 3 && blockId == 0x20 {
			// before 1.03, 0x20 was used instead of 0x22
			blockId = 0x22
		}

		switch blockId {
		case 0x00:
			// normally zeroes signify the end of the replay
			// but I want to make sure there aren't zeroes in between blocks
			zeroes++
		case 0x17: // LeaveGame
			numLeaves++
			reason, err := readDWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading leavegame reason")
			}
			playerId, err := buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading leavegame playerId")
			}
			result, err := readDWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading leavegame result")
			}
			unknown, err := readDWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading leavegame unknown val")
			}
			inc := unknown > leaveUnknown
			leaveUnknown = unknown
			playerRecords[int(playerId)].leaveTime = currentTimeMS
			curPlayer := rep.Players[int(playerId)]

			// last leave action is by the saver
			if numLeaves == len(rep.Players) {
				rep.Saver = curPlayer
				if saverWon {
					rep.WinnerTeam = rep.Saver.slot.TeamNumber
				}
			}

			// TODO: maybe store all the losers to help deduce the winner
			// until then, we only care about win conditions
			switch reason {
			case 0x01, 0x0E:
				switch result {
				case 0x09:
					rep.WinnerTeam = curPlayer.slot.TeamNumber
				}
			case 0x0C:
				if rep.Saver == nil { // "not last"
					switch result {
					case 0x09:
						saverWon = true
					case 0x0A:
						rep.WinnerTeam = -1 // draw
					}
				} else { // last local leave action => curPlayer == rep.Saver
					switch result {
					case 0x07, 0x0B:
						if inc {
							rep.WinnerTeam = rep.Saver.slot.TeamNumber
						}
					case 0x09:
						rep.WinnerTeam = rep.Saver.slot.TeamNumber
					}
				}
			}

		case 0x1A: //first startblock
			if err := expectDWORD(buffer, 0x01); err != nil {
				return fmt.Errorf("error reading first startblock: %w", err)
			}
		case 0x1B: //second startblock
			if err := expectDWORD(buffer, 0x01); err != nil {
				return fmt.Errorf("error reading second startblock: %w", err)
			}
		case 0x1C: //third startblock
			if err := expectDWORD(buffer, 0x01); err != nil {
				return fmt.Errorf("error reading third startblock: %w", err)
			}
		case 0x1E, 0x1F: // time slot
			timeSlotLen, err := readWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading timeslot block len: %w", err)
			}
			if ms, err := readWORD(buffer); err != nil {
				return fmt.Errorf("error reading timeslot time increment: %w", err)
			} else {
				currentTimeMS += int(ms)
			}
			if timeSlotLen <= 2 {
				break
			}

			commandDataBlock := make([]byte, timeSlotLen-2)
			if _, err := buffer.Read(commandDataBlock); err != nil {
				return fmt.Errorf("error reading commanddata block: %w", err)
			}
			commandDataBuf := bytes.NewBuffer(commandDataBlock)
			for commandDataBuf.Len() > 0 {
				var player *Player
				if playerId, err := commandDataBuf.ReadByte(); err != nil {
					return fmt.Errorf("error reading CommandData playerId: %w", err)
				} else {
					player = rep.Players[int(playerId)]
				}
				actionBlockLen, err := readWORD(commandDataBuf)
				if err != nil {
					return fmt.Errorf("error reading action block len: %w", err)
				}
				actionBlockBytes := make([]byte, actionBlockLen)
				if _, err := commandDataBuf.Read(actionBlockBytes); err != nil {
					return fmt.Errorf("error reading action block: %w", err)
				}
				actionBlockBuf := bytes.NewBuffer(actionBlockBytes)
				for actionBlockBuf.Len() > 0 {
					actionable, err := readActionBlock(actionBlockBuf, rep)
					if err != nil {
						if err == io.EOF {
							// FIXME: this is just so we don't crash from unimplemented actions
							break
						}
						return fmt.Errorf("error parsing action block: %w", err)
					}
					action := Action{
						Ability: actionable,
						Time:    time.Duration(currentTimeMS) * time.Millisecond,
						Player:  player,
					}
					if action.Ability != nil {
						rep.Actions = append(rep.Actions, action)
					}
				}
			}

		case 0x20: //chat message
			playerId, err := buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading chat message playerId: %w", err)
			}
			_, err = readWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading chat message block len: %w", err)
			}
			flags, err := buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading chat message block flags: %w", err)
			}
			var dest MsgDestination
			if flags != 0x10 {
				chatMode, err := readDWORD(buffer)
				if err != nil {
					return fmt.Errorf("error reading chat message block mode: %w", err)
				}
				switch chatMode {
				case 0x00:
					dest = MsgToEveryone{}
				case 0x01:
					dest = MsgToAllies{}
				case 0x02:
					dest = MsgToObservers{}
				default:
					dest = MsgToPlayer{*rep.Players[int(chatMode)-2].slot}
				}
			}

			msg, err := buffer.ReadString(0)
			if err != nil {
				return fmt.Errorf("error reading msg text: %w", err)
			}
			msg = strings.TrimRight(msg, "\000")
			timestamp := time.Duration(currentTimeMS) * time.Millisecond

			rep.ChatMessages = append(rep.ChatMessages, ChatMessage{
				Timestamp:   timestamp,
				Author:      *rep.Players[int(playerId)].slot,
				Body:        msg,
				Destination: dest,
			})
		case 0x22: //checksum?
			n, err := buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading checksum block len: %w", err)
			}
			buffer.Next(int(n))
		case 0x23: //unknown
			buffer.Next(10)
		case 0x2F: // forced game end countdown (map is revealed)
			mode, err := readDWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading game end cd mode: %w", err)
			}
			countdownSecs, err := readDWORD(buffer)
			if err != nil {
				return fmt.Errorf("error reading game end cd secs: %w", err)
			}
			// TODO
			if rep.debugMode {
				fmt.Printf("countdown mode %x, %d\n", mode, countdownSecs)
			}
		default:
			if rep.debugMode {
				fmt.Printf("unknown block id: 0x%X\n", blockId)
			}
		}
		if blockId != 0 && zeroes > 0 {
			fmt.Println(zeroes, "zeroes")
			zeroes = 0
		}

	}

	return nil
}

func readBattleNetAcct(bnetBuffer *bytes.Buffer) (account BattleNet2Account, err error) {
	if err := expectByte(bnetBuffer, 0x0A); err != nil {
		return account, err
	}

	// then, the length of this account entry
	bnetAccountBlockLength, err := bnetBuffer.ReadByte()
	if err != nil {
		return account, fmt.Errorf("error reading block length: %w", err)
	}
	bnetAccountBlock := make([]byte, bnetAccountBlockLength)
	_, err = bnetBuffer.Read(bnetAccountBlock)
	if err != nil {
		return account, fmt.Errorf("error reading account block: %w", err)
	}

	bnetAccountBuffer := bytes.NewBuffer(bnetAccountBlock)
	return readBnetAcctInner(bnetAccountBuffer)
}
func readBnetAcctInner(bnetAccountBuffer *bytes.Buffer) (account BattleNet2Account, err error) {
	for bnetAccountBuffer.Len() > 0 {
		sectionByte, err := bnetAccountBuffer.ReadByte()
		if err != nil {
			return account, fmt.Errorf("error reading account block's section: %w", err)
		}
		switch sectionByte {
		case 0x08: // id of playerRecord
			pRecId, err := bnetAccountBuffer.ReadByte()
			if err != nil {
				return account, fmt.Errorf("error reading account's playerRecord id: %w", err)
			}
			account.PlayerId = int(pRecId)

			break
		case 0x12: // username
			bnetUsername, err := readLengthAndThenString(bnetAccountBuffer)
			if err != nil {
				return account, fmt.Errorf("error reading username: %w", err)
			}
			account.Username = bnetUsername
			break
		case 0x22: // avatar
			avatarName, err := readLengthAndThenString(bnetAccountBuffer)
			if err != nil {
				return account, fmt.Errorf("error reading avatar: %w", err)
			}
			account.Avatar = avatarName
			break
		case 0x1A: // this seems to always just be the string "clan"
			clanName, err := readLengthAndThenString(bnetAccountBuffer)
			if err != nil {
				return account, fmt.Errorf("error reading clan: %w", err)
			}
			account.Clan = clanName
		case 0x28: // no idea what this represents
			account.ExtraData, _ = ioutil.ReadAll(bnetAccountBuffer)

		default:
			fmt.Printf("[***] Unrecognized byte in block: 0x%x\n", sectionByte)
		}
	}
	if account.Avatar == "" {
		account.Avatar = "p003" // make avatar peon by default, as that is the ingame default
	}
	return account, nil
}

// parses the "encoded string" part of the data
func readEncodedString(encodedStr string, rep *Replay) error {
	decodedBytes := decodeString(encodedStr)
	decoded := bytes.NewBuffer(decodedBytes)
	gameSpeedFlag, err := decoded.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading game speed: %w", err)
	}

	rep.GameOptions.Speed = GameSpeed(gameSpeedFlag)
	byte2, err := decoded.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading game settings: %w", err)
	}
	visibilityBits := byte2 & 0b1111
	visibility := Visibility(bits.LeadingZeros8(visibilityBits) - 4)
	rep.GameOptions.Visibility = visibility
	observerBits := (byte2 >> 4) & 0b11
	rep.GameOptions.ObserverSetting = ObserverSetting(observerBits)

	teamsTogether := ((byte2 >> 6) & 1) == 1
	rep.GameOptions.TeamsTogether = teamsTogether

	fixedTeamsByte, err := decoded.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading game settings: %w", err)
	}
	fixedTeamsByte = fixedTeamsByte >> 1
	lockTeams := (fixedTeamsByte & 0b11) == 3
	rep.GameOptions.LockTeams = lockTeams

	byte3, err := decoded.ReadByte()
	if err != nil {
		return fmt.Errorf("error reading game settings: %w", err)
	}
	fullSharedUnitControl := byte3&1 == 1
	randomHero := (byte3>>1)&1 == 1
	randomRaces := (byte3>>2)&1 == 1
	observerReferees := (byte3>>6)&1 == 1
	rep.GameOptions.FullSharedUnitControl = fullSharedUnitControl
	rep.GameOptions.RandomHero = randomHero
	rep.GameOptions.RandomRaces = randomRaces
	if observerReferees {
		rep.GameOptions.ObserverSetting = ObsReferees
	}
	_, _ = decoded.Read(make([]byte, 5+4)) //skip unknown bytes & map checksum (4)
	mapName, err := decoded.ReadString(0)
	if err != nil {
		return fmt.Errorf("error reading map name: %w", err)
	}

	mapName = strings.ReplaceAll(mapName, "\\", "/")

	rep.GameOptions.MapName = strings.TrimRight(mapName, "\000")
	gameCreatorName, err := decoded.ReadString(0)
	if err != nil {
		return fmt.Errorf("error reading game creator name: %w", err)
	}
	rep.GameOptions.CreatorName = strings.TrimRight(gameCreatorName, "\000")

	if s, err := decoded.ReadString(0); err != nil {
		return err
	} else if s != "\000" {
		return fmt.Errorf("third decoded string should have been empty: '%s'", s)
	}
	return nil
}
func readPlayerRecord(buffer *bytes.Buffer, rep *Replay) (playerRecord playerRecord, err error) {

	playerId, err := buffer.ReadByte()
	if err != nil {
		return playerRecord, fmt.Errorf("error reading player id: %w", err)
	}
	playerRecord.Id = int(playerId)
	playerName, err := buffer.ReadString(0) // read null terminated string
	if err != nil {
		return playerRecord, fmt.Errorf("error reading player name: %w", err)
	}
	playerRecord.Name = strings.TrimSuffix(playerName, "\000")
	additionalDataSize, err := buffer.ReadByte()
	if err != nil {
		return playerRecord, fmt.Errorf("error reading player additional data size: %w", err)
	}
	// seems to be 0 in reforged
	if additionalDataSize == 0x1 {
		// skip null byte
		if err = expectByte(buffer, 0); err != nil {
			return playerRecord, err
		}
	} else if additionalDataSize == 0x8 {
		// For ladder games only
		// TODO: make use of these
		// runtime of players Warcraf.exe in milliseconds
		if runtimeMS, err := readDWORD(buffer); err != nil {
			return playerRecord, fmt.Errorf("error reading player exe runtime: %w", err)
		} else {
			//fmt.Println(runtimeMS)
			playerRecord.RuntimeMS = runtimeMS
		}
		// player race flags:
		if playerRaceFlags, err := readDWORD(buffer); err != nil {
			return playerRecord, fmt.Errorf("error reading player race flags: %w", err)
		} else {
			if rep.debugMode {
				fmt.Printf("player race flag: 0x%x\n", playerRaceFlags)
			}
			playerRecord.RaceFlags = playerRaceFlags
		}
	} else if additionalDataSize != 0 {
		if rep.debugMode {
			fmt.Printf("Warning: unrecognized additional data size: 0x%x\n", additionalDataSize)
		}
		additionalData := make([]byte, additionalDataSize)
		_, err = buffer.Read(additionalData)
		if rep.debugMode {
			fmt.Println("additional data:", additionalData)
		}
	}
	return
}

// internal struct for gathering together data about a player as we move through the file
// playerRecord is the first thing about the player we encounter
// so it's a good starting point to later append bnet and slot stuff to
type playerRecord struct {
	Id        int
	Name      string
	RuntimeMS uint32
	RaceFlags uint32
	Bnet2Acc  *BattleNet2Account
	SlotId    int
	leaveTime int
}

type header struct {
	GameVersion    int
	HeaderVersion  uint32
	NumberOfBlocks uint32
	Length         uint32
	BuildNumber    int
	IsMultiplayer  bool
	Duration       time.Duration
	Expac          Expac
}
