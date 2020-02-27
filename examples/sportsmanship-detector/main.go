package main

import (
	"fmt"
	"github.com/efskap/warcrumb"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s ReplayToCheck.w3g\n", os.Args[0])
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	replay, err := warcrumb.ParseReplay(f)
	if err != nil {
		log.Fatal("error parsing replay: ", err)
	}
	sportsmanTerms := []string{"gg", "glhf"}

	isSportsmanlike := make(map[*warcrumb.Player]bool)
	for _, msg := range replay.ChatMessages {
		for _, term := range sportsmanTerms {
			if strings.Contains(strings.ToLower(msg.Body), term) {
				isSportsmanlike[msg.Author] = true
			}
		}
	}
	for _, player := range replay.Players {
		if isSportsmanlike[player] {
			fmt.Println(player, "has demonstrated sportsmanship! Well done!")
		} else {
			fmt.Println(player, "has NOT been sportsmanlike this game. BOOOOO!")
			fmt.Printf("Typical %s player...\n", replay.Slots[player.SlotId].Race)
		}
	}
}
