package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	f, err := os.Open("testReplays/reforgedPudgeWars.w3g")
	//f, err := os.Open("testReplays/1.18-replayspl_4105_MKpowa_KrawieC..w3g")
	//f, err := os.Open("reforgedPudgeWars.w3g")
	//f, err := os.Open("./FirstWin.w3g")
	//f, err := os.Open("./1.18-replayspl_4105_MKpowa_KrawieC..w3g")
	//f, err := os.Open("./1.01-LeoLaporte_vs_Ghostridah_crazy.w3g")
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
	error := json.Indent(&prettyJSON, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)
		return
	}

	fmt.Println(string(prettyJSON.Bytes()))

}
