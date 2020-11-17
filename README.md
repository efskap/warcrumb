# warcrumb
## Warcraft 3 Replay Parser in Go

![Go](https://github.com/efskap/warcrumb/workflows/Go/badge.svg)

A work in progress, much like WC3 Reforged.

Pulls out metadata, chat, and game events (to some extent).

Supports all versions of the game, including Reforged.

Based on http://w3g.deepnode.de/files/w3g_format.txt, with my own research into the Reforged format (e.g. Battle.net 2.0 integration).

## A word on what is encoded

Please note that a WC3 replay does not record what _happened_, but rather what inputs the human players sent. The game (inc. AI) is deterministic, so it can be recreated from this.

This means you can't really know:

- what the AI did
- combat stuff like which units died (at best you can infer this from selections), resources at a given moment, etc 
- what _actually happened_ (e.g. you can have a "build tower" ability encoded, even if it was cancelled thereafter)


## Loading a replay

```go
f, err := os.Open("LastReplay.w3g")
if err != nil {
    log.Fatal(err)
}
replay, err := warcrumb.ParseReplay(f)
if err != nil {
    log.Fatal("error parsing replay: ", err)
}
```

### Example: Actions

```go
for _, action := range replay.Actions {
    if ability, ok := action.Ability.(warcrumb.BasicAbility); ok {
        fmt.Println(fmtTimestamp(action.Time), action.Player, ability.ItemId.String())
    } else if ability, ok := action.Ability.(warcrumb.TargetedAbility); ok {
        fmt.Println(fmtTimestamp(action.Time), action.Player, ability.ItemId.String(), ability.Target)
    }
}

// ---

func fmtTimestamp(duration time.Duration) string {
	mins := duration.Truncate(time.Minute)
	secs := (duration - mins).Truncate(time.Second).Seconds()
	return fmt.Sprintf("%02d:%02d", int(mins.Minutes()), int(secs))
}

```

Prints stuff like:
```
13:34 LeoLaporte Train Raider
13:38 Ghostridah. Build Beastiary (-6592.0,-2240.0)
13:40 Ghostridah. Train Wind Rider 
```

### Example: Sportsmanship Chat Analyzer

```go
sportsmanTerms := []string{"gg", "glhf"}
isSportsmanlike := make(map[*warcrumb.Player]bool)

for _, msg := range replay.ChatMessages {
    for _, term := range sportsmanTerms {
        if strings.Contains(strings.ToLower(msg.Body), term) {
            isSportsmanlike[msg.Author.Player] = true
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
```

See [`/examples`](examples) for the full programs and other examples.
