package main

import (
	"flag"
	"fmt"
	"github.com/efskap/warcrumb"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Prints what the slots on the lobby screen would look like
// (name, race, team, color, handicap)
func main() {
	colorterm := os.Getenv("COLORTERM")
	termSupportsColor := colorterm == "truecolor" || colorterm == "24bit"
	useColor := flag.Bool("color", termSupportsColor, "show player colors as boxes (requires truecolor term)")
	flag.Usage = func() {
		fmt.Printf("Usage: %s LastReplay.w3g ...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	for i, arg := range flag.Args() {
		f, err := os.Open(arg)
		if err != nil {
			log.Fatalf("cannot open %s: %s", arg, err)
		}
		replay, err := warcrumb.ParseReplay(f)
		if err != nil {
			log.Fatalf("cannot parse %s: %s", arg, err)
		}
		fmt.Println(replay.GameOptions.GameName)
		fmt.Println(filepath.Base(replay.GameOptions.MapName))
		printLobby(replay, *useColor)
		if i < flag.NArg()-1 {
			fmt.Println(strings.Repeat("\u2015", 60))
		}
	}

}

func printLobby(replay warcrumb.Replay, useColor bool) {
	for _, slot := range replay.Slots {
		fmt.Printf("%-20s", slot.String())
		if slot.SlotStatus == warcrumb.UsedSlot {
			var colouredBox string
			if useColor {
				colouredBox = fmt.Sprint(setFgColor(slot.Color), "\u2588\u2588", resetColor())
			}
			fmt.Printf("\t%-11s\tTeam %d\t%s\t%d%%",
				slot.Race.String(),
				slot.TeamNumber+1,
				colouredBox,
				slot.Handicap,
			)
			if replay.WinnerTeam == slot.TeamNumber {
				fmt.Print("\t(winner)")
			}
			if slot.Player == replay.Saver {
				fmt.Print("\t(saver)")
			}
		}
		fmt.Println()
	}
}
func setFgColor(color color.Color) string {
	r, g, b, _ := color.RGBA()
	return fmt.Sprintf("\x01\x1b[38;2;%d;%d;%dm\x02", r, g, b)
}
func resetColor() string {
	return "\x01\x1b[0m\x02"
}
