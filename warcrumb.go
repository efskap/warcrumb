package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

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
	//f, err := os.Open("testReplays/1.18-replayspl_4105_MKpowa_KrawieC..w3g")
	//f, err := os.Open("reforgedPudgeWars.w3g")
	//f, err := os.Open("./FirstWin.w3g")
	//f, err := os.Open("./1.18-replayspl_4105_MKpowa_KrawieC..w3g")
	//f, err := os.Open("./1.01-LeoLaporte_vs_Ghostridah_crazy.w3g")

}
func printJson(filepath string) {

	f, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rep, err := Read(f)
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
