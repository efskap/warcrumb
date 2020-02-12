package main

import (
	"flag"
	"fmt"
	"github.com/efskap/warcrumb"
	"image/color"
	"log"
	"os"
	"strings"
)

// Prints what the slots on the lobby screen would look like
// (name, race, team, color, handicap)
// requires a truecolor terminal for colors
func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Printf("Usage: %s LastReplay.w3g ...\n", os.Args[0])
		flag.PrintDefaults()
	}
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
		printLobby(replay)
		if i < flag.NArg()-1 {
			fmt.Println(strings.Repeat("\u2015", 60))
		}
	}

}

func printLobby(replay warcrumb.Replay) {
	for _, slot := range replay.Slots {
		fmt.Print(paddedStr(slot.NameText(), 20))
		if slot.SlotStatus == warcrumb.UsedSlot {
			fmt.Print("\t",
				paddedStr(slot.Race.String(), 11), "\t",
				"Team ", slot.TeamNumber+1, "\t",
				setFgColor(slot.Color), "\u2588\u2588", resetColor(), "\t", // colored box
				slot.Handicap, "%",
			)
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
func paddedStr(str string, i int) string {
	if len(str) >= i {
		return ""
	}
	return str + strings.Repeat(" ", i-len(str))
}
