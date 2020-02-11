package main

import (
	"flag"
	"fmt"
	"github.com/efskap/warcrumb"
	"image/color"
	"log"
	"os"
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
	for _, arg := range flag.Args() {
		f, err := os.Open(arg)
		if err != nil {
			log.Fatalf("cannot open %s: %s", arg, err)
		}
		replay, err := warcrumb.ParseReplay(f)
		if err != nil {
			log.Fatalf("cannot parse %s: %s", arg, err)
		}
		printLobby(replay)
	}

}

func printLobby(replay warcrumb.Replay) {
	for _, slot := range replay.Slots {
		fmt.Print(slot.NameText())
		for i := 0; i < 16-len(slot.NameText()); i++ {
			fmt.Print(" ")
		} // pad out name
		if slot.SlotStatus == warcrumb.UsedSlot {
			fmt.Print("\t", slot.Race.String())
			for i := 0; i < 11-len(slot.Race.String()); i++ {
				fmt.Print(" ")
			} // pad out race
			fmt.Print("\t",
				"Team ", slot.TeamNumber+1, "\t",
				setFgColor(slot.Color), "\u2588\u2588", resetColor(), "\t",
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
