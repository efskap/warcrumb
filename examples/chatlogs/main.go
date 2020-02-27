package main

import (
	"flag"
	"fmt"
	"github.com/efskap/warcrumb"
	"image/color"
	"log"
	"os"
	"strings"
	"time"
)

// Prints a replay's chat messages
func main() {
	colorterm := os.Getenv("COLORTERM")
	termSupportsColor := colorterm == "truecolor" || colorterm == "24bit"
	useColor := flag.Bool("color", termSupportsColor, "print player names in their respective colors")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS] LastReplay.w3g ...\n", os.Args[0])
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
		printChat(replay, *useColor)
		if i < flag.NArg()-1 {
			fmt.Println(strings.Repeat("\u2015", 60))
		}
	}

}

func printChat(replay warcrumb.Replay, color bool) {
	for _, msg := range replay.ChatMessages {
		playerName := msg.Author.String()
		if color {
			playerName = fmt.Sprint(setFgColor(replay.Slots[msg.Author.SlotId].Color), msg.Author, resetColor())
		}
		fmt.Printf("[%s] [%s] %s: %s\n", fmtTimestamp(msg.Timestamp), msg.Destination, playerName, msg.Body)
	}
}
func setFgColor(col color.Color) string {
	rgb := color.RGBAModel.Convert(col).(color.RGBA)
	return fmt.Sprintf("\x01\x1b[38;2;%d;%d;%dm\x02", rgb.R, rgb.G, rgb.B)
}
func resetColor() string {
	return "\x01\x1b[0m\x02"
}

func fmtTimestamp(duration time.Duration) string {
	mins := duration.Truncate(time.Minute)
	secs := (duration - mins).Truncate(time.Second).Seconds()
	return fmt.Sprintf("%02d:%02d", int(mins.Minutes()), int(secs))
}
