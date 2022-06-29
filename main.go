//--Summary:
//  Create a grep clone that can do simple substring searching
//  within files. It must auto-recurse into subdirectories.
//
//--Requirements:
//* Use goroutines to search through the files for a substring match
//* Display matches to the terminal as they are found
//  * Display the line number, file path, and complete line containing the match
//* Recurse into any subdirectories looking for matches
//* Use any synchronization method to ensure that all files
//  are searched, and all results are displayed before the program
//  terminates.
//
//--Notes:
//* Program invocation should follow the pattern:
//    mgrep search_string search_dir

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FoundString struct {
	path  string
	index int
	line  string
}

var wg sync.WaitGroup
var found = make(chan FoundString)
var results = make(chan []FoundString)
var errors = make(chan error)

func check(e error) {
	if e != nil {
		errors <- e
	}
}

func (fs FoundString) String() string {
	result := "Hit found in file " + fs.path + "\n"
	result += fmt.Sprintf("Line %v: \n", fs.index)
	result += fs.line + "\n"

	return result
}

func collectStrings() {
	var list []FoundString
	for {
		value, ok := <-found
		if !ok {
			break
		}
		time.Sleep(50 * time.Millisecond)
		list = append(list, value)
	}
	if len(list) > 0 {
		results <- list
	}
}

func parseFile(path, searchString string) {
	defer wg.Done()

	fileData, err := os.ReadFile(path)
	check(err)

	lines := strings.Split(string(fileData), "\n")

	for index, line := range lines {
		if strings.Contains(line, searchString) {
			found <- FoundString{
				path:  path,
				index: index,
				line:  line,
			}
		}
	}

}

func parseDir(path, searchString string) {
	defer wg.Done()

	directoryList, err := os.ReadDir(path)
	check(err)

	for _, file := range directoryList {
		fileInfo, err := file.Info()
		check(err)
		completePath := filepath.Join(path, fileInfo.Name())
		if file.IsDir() {
			wg.Add(1)
			go parseDir(completePath, searchString)
			continue
		}
		wg.Add(1)
		go parseFile(completePath, searchString)
	}

}

func main() {
	// Fire service for collecting results
	go collectStrings()
	searchString := os.Args[1]
	searchDir := os.Args[2]

	wg.Add(1)
	go parseDir(searchDir, searchString)

	wg.Wait()
	close(found)
	hits := <-results
	for _, hit := range hits {
		fmt.Println(hit)
	}

}
