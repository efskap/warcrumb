package main

import (
	"fmt"
	"github.com/efskap/warcrumb"
	"log"
	"os"
	"time"
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

	for _, action := range replay.Actions {
		if ability, ok := action.Ability.(warcrumb.BasicAbility); ok {
			fmt.Println(fmtTimestamp(action.Time), action.Player, ability.ItemId.String())
		} else if ability, ok := action.Ability.(warcrumb.TargetedAbility); ok {
			fmt.Println(fmtTimestamp(action.Time), action.Player, ability.ItemId.String(), ability.Target)
		}
	}
}

func fmtTimestamp(duration time.Duration) string {
	mins := duration.Truncate(time.Minute)
	secs := (duration - mins).Truncate(time.Second).Seconds()
	return fmt.Sprintf("%02d:%02d", int(mins.Minutes()), int(secs))
}
