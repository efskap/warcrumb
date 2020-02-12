package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/efskap/warcrumb"
	"log"
	"os"
)

// Just dump the replay as a JSON object... for testing mostly
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
		printJson(arg)
	}

}
func printJson(filepath string) {

	f, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rep, err := warcrumb.ParseReplay(f)
	if err != nil {
		log.Println(err)
	}
	body, err := json.Marshal(rep)
	if err != nil {
		log.Println(err)
	}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "\t")
	if err != nil {
		log.Println("JSON parse error: ", err)
		return
	}

	fmt.Println(string(prettyJSON.Bytes()))
}
