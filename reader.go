package main

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

func Read(file io.Reader) (rep Replay, err error) {
	header, err := readHeader(file)
	if err != nil {
		return rep, fmt.Errorf("error reading header: %w", err)
	}
	rep.IsMultiplayer = header.IsMultiplayer
	rep.Duration = header.Duration
	rep.Version = header.GameVersion
	rep.Expac = header.Expac
	rep.BuildNumber = header.BuildNumber

	rep.isReforged = rep.Version >= 10000

	// might as well allocate the right size buffer based on the assumption that every block is 8K
	buffer := bytes.NewBuffer(make([]byte, 0, header.NumberOfBlocks*0x2000))
	for i := 0; i < int(header.NumberOfBlocks); i++ {
		b, err := readCompressedBlock(file, rep.isReforged)
		if err != nil {
			return rep, fmt.Errorf("failed to decompress block i=%d: %w", i, err)
		}
		buffer.Write(b)
	}

	err = readDecompressedData(buffer, &rep)
	if err != nil {
		return rep, fmt.Errorf("error in decompressed data: %w", err)
	}

	_ = os.Mkdir("hexdumps", os.ModePerm)
	_ = ioutil.WriteFile(fmt.Sprintf("./hexdumps/decompresssed_%s_%s.hex",rep.GameOptions.GameName, rep.GameOptions.CreatorName), buffer.Bytes(), os.ModePerm)
	if buffer.Len() > 0 {
		fmt.Printf("[*] %d unread bytes left!!!\n\n", buffer.Len())
	}
	return
}

func readHeader(file io.Reader) (header Header, err error) {
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

	compressedFileSize, err := readDWORD(file)
	if err != nil {
		return header, fmt.Errorf("error reading compressed file size: %w", err)
	}

	fmt.Println(compressedFileSize)

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
	playerRecords := make(map[int]playerRecord)
	// [playerRecord]
	if err = expectByte(buffer, 0); err != nil {
		return err
	}
	p, err := readPlayerRecord(buffer)
	if err != nil {
		return err
	}
	playerRecords[p.Id] = p

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
		str = strings.TrimRight(string(append([]byte{b}, str...)), "\000")
		fmt.Println("mystery string:", str)
	}

	encodedString, err := buffer.ReadString(0) // read null terminated string
	if err != nil {
		return fmt.Errorf("error reading encoded string: %w", err)
	}

	if err = readEncodedString(encodedString, rep); err != nil {
		return fmt.Errorf("error reading decoded string: %w", err)
	}

	// [PlayerCount]
	playerCount, err := readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading player count: %w", err)
	}
	//rep.Slots = make([]Slot, playerCount)

	fmt.Println("playercount", playerCount)
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
	fmt.Printf("private flag: 0x%x\n", privateFlag)
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

	unknownMaybeLangId, err := readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading LanguageID: %w", err)
	}
	fmt.Printf("LanguageID (?) = 0x%x\n", unknownMaybeLangId)

	// player record, bnet, break on gamestartrecord
	for {
		recordId, err := buffer.ReadByte()
		if err != nil {
			return fmt.Errorf("error reading record id: %w", err)
		}
		//fmt.Printf("RecordID: 0x%x\n", recordId)

		if recordId == 0x16 {
			// playerRecord
			if pRec, err := readPlayerRecord(buffer); err != nil {
				return err
			} else {
				playerRecords[pRec.Id] = pRec
			}
			if _, err = readDWORD(buffer); err != nil {
				return err
			}
		} else if recordId == 0x39 {
			fmt.Println("[*] Battle.net 2.0 data present")

			// not sure what these next 7 bytes are but they seem consistent?
			restOfBnetMagic := make([]byte, 7)
			_, err := buffer.Read(restOfBnetMagic)
			if err != nil {
				return fmt.Errorf("error reading rest of bnet2.0 magic: %w", err)
			}

			if !bytes.Equal(restOfBnetMagic, []byte{4, 0, 0, 0, 0, 57, 3}) {
				fmt.Printf("wasn't expecting that magic string: %v\n", restOfBnetMagic)
			}

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
			_ = ioutil.WriteFile("bnetBlock.hex", bnetBlock, os.ModePerm)
			bnetBuffer := bytes.NewBuffer(bnetBlock)

			// now we don't know how many account entries are in the block
			// but we know the size of the whole thing, so just read it until it's empty
			// each iteration reads one account
			for bnetBuffer.Len() > 0 {
				// first, 0x0A
				acct, err := readBattleNetAcct(bnetBuffer)
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

		} else if recordId == 0x19 {
			break
		} else {
			fmt.Printf("Not sure how to handle recordId 0x%x\n", recordId)
			break
		}
	}

	// GameStartRecord

	if dataBytes, err := readWORD(buffer); err != nil {
		return err
	} else {
		fmt.Println(dataBytes, "data bytes")
	}
	nr, err := buffer.ReadByte()
	if err != nil {
		return err
	} else {
		fmt.Println(nr, "slot records")
	}
	rep.Slots = make([]Slot, 0, nr)
	for slotId := 0; slotId < int(nr); slotId++ {
		playerId, err := buffer.ReadByte()
		if err != nil {
			return err
		}
		pRec, ok := playerRecords[int(playerId)]
		pRec.SlotId = slotId

		var slotRecord Slot
		if playerId != 0 {
			if ! ok {
				return fmt.Errorf("slot references invalid player record: id=%d", playerId)
			} else {
				playerRecords[int(playerId)] = pRec
			}
		}
			//fmt.Printf("slot references invalid player record: id=%d\n", playerId)
		slotRecord.PlayerId = pRec.Id

		if mapDownloadPct, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.MapDownloadPercent = mapDownloadPct
			if !( mapDownloadPct == 255 || mapDownloadPct == 100 ) {
				return fmt.Errorf("sanity check failed: playerId = %d, map download %% = 0x%x", playerId, mapDownloadPct)
			}
		}
		if slotStatus, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.SlotStatus = SlotStatus(slotStatus)
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
			slotRecord.TeamNumber = int(teamNumber)
		}
		if color, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			slotRecord.Color = Color(color)
		}

		if playerRace, err := buffer.ReadByte(); err != nil {
			return err
		} else {
			if playerRace&0x40 > 0 {
				slotRecord.raceSelectableOrFixed = true
			}
			slotRecord.Race = races[playerRace]
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

	randomSeed, err := readDWORD(buffer)
	if err != nil {
		return fmt.Errorf("error reading random seed: %w", err)
	}
	rep.RandomSeed = randomSeed

	// select mode




	rep.Players = make([]Player, 0, len(playerRecords))
	for id, pRec := range playerRecords {
		if id != pRec.Id  || rep.Slots[pRec.SlotId].PlayerId != id{
			return fmt.Errorf("ID WAS NOT SET CORRECTLY")

		}
		rep.Players = append(rep.Players, Player{
			SlotId:  pRec.SlotId,
			BattleNet: pRec.Bnet2Acc,
			Name: pRec.Name,
		})
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
func readPlayerRecord(buffer *bytes.Buffer) (playerRecord playerRecord, err error) {

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
			fmt.Println(runtimeMS)
			playerRecord.RuntimeMS = runtimeMS
		}
		// player race flags:
		if playerRaceFlags, err := readDWORD(buffer); err != nil {
			return playerRecord, fmt.Errorf("error reading player race flags: %w", err)
		} else {
			fmt.Printf("player race flag: 0x%x\n", playerRaceFlags)
			playerRecord.RaceFlags = playerRaceFlags
		}
	} else if additionalDataSize != 0 {
		fmt.Printf("Warning: unrecognized additional data size: 0x%x\n", additionalDataSize)
		additionalData := make([]byte, additionalDataSize)
		_, err = buffer.Read(additionalData)
		fmt.Println("additional data:", additionalData)
	}
	return
}

// internal struct for gathering together data about a player as we move through the file
// playerRecord is the first thing about the player we encounter
// so it's a good starting point to later append bnet and slot stuff to
type playerRecord struct {
	Id         int
	Name       string
	RuntimeMS  uint32
	RaceFlags  uint32
	Bnet2Acc   *BattleNet2Account
	SlotId     int
}

type Header struct {
	GameVersion    int
	HeaderVersion  uint32
	NumberOfBlocks uint32
	Length         uint32
	BuildNumber    int
	IsMultiplayer  bool
	Duration       time.Duration
	Expac          Expac
}
