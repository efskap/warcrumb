// This program generates strings.go from a directory containing *strings.txt extracted from the WC3 mpq/casc
package main

import (
	"bufio"
	"fmt"
	. "github.com/efskap/warcrumb"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var headerRegex = regexp.MustCompile(`^\[(\w{4})\]$`)
var attrRegex = regexp.MustCompile(`^(\w+)=(.*)$`)

const targetPath = "strings.go"
const header = `// Code generated by gen_strings. DO NOT EDIT.`
const tmpl = header + `
package warcrumb

var WC3Strings=`

func main() {
	if len(os.Args) != 2 {
		printUsage()
		os.Exit(1)
	}

	stringsDir := os.Args[1]

	entityMap := make(map[string]StringsEntity)
	files, err := filepath.Glob(filepath.Join(stringsDir, "*strings.txt"))
	if err != nil {
		log.Fatalf("error scanning dir %s: %s", stringsDir, err)
	}
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var currentEntity *StringsEntity
		for scanner.Scan() {
			if matches := headerRegex.FindStringSubmatch(scanner.Text()); len(matches) > 1 {
				currentEntity = &StringsEntity{Code: matches[1]}
			}
			if currentEntity != nil {
				if matches := attrRegex.FindStringSubmatch(scanner.Text()); len(matches) > 2 {
					val := matches[2]
					switch matches[1] {
					case "Name":
						currentEntity.Name = val
					case "Tip":
						currentEntity.Tip = val
					}
				}
				entityMap[currentEntity.Code] = *currentEntity
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("error reading %s: %s", path, err)
		}
	}

	if _, err = os.Stat(targetPath); !os.IsNotExist(err) {
		contents, err2 := ioutil.ReadFile(targetPath)
		if err2 != nil {
			log.Fatal(err2)
		}
		if !strings.Contains(string(contents), header) {
			log.Fatalf("%s is not empty, and it doesn't appear to be generated by gen_strings. Aborting.", targetPath)
		}
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	mapRepr := fmt.Sprintf("%#v", entityMap)
	// no need to specify package name since we'll be inside it
	mapRepr = strings.ReplaceAll(mapRepr, "warcrumb.", "")

	_, err = fmt.Fprintln(f, tmpl, mapRepr)
	if err != nil {
		log.Fatalf("error writing to %s: %s", targetPath, err)
	}
	fmt.Println("generated", targetPath)
}
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s path/to/strings/folder/", os.Args[0])
}
