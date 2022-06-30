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
)

type FoundString struct {
	path  string
	index int
	line  string
}

var wg sync.WaitGroup
var found = make(chan FoundString)

// set the capacity to 1 so we can write the results
// to the channel without having a reader on the other end
var results = make(chan []FoundString, 1)
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

func collectStrings(lock *sync.Mutex) {
	defer lock.Unlock()

	// As long as this goroutine is running, we want the lock to be held.
	// This will be used later
	lock.Lock()

	var list []FoundString
	for {
		value, ok := <-found
		if !ok {
			// main thread closes channel indicating that search is complete
			break
		}
		// sleep isn't needed
		// time.Sleep(50 * time.Millisecond)
		list = append(list, value)
	}

	// once we break from the above loop, we send the results on the channel
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
	// Lock is used for the collectStrings goroutine. As long as the
	// goroutine is running, then the lock will be taken. When the
	// lock is released, then this is an indication that the results
	// have been compiled properly, and they can be read frmo the channel.
	var resultsLock = sync.Mutex{}

	// Fire service for collecting results
	go collectStrings(&resultsLock)

	searchString := os.Args[1]
	searchDir := os.Args[2]

	wg.Add(1)
	go parseDir(searchDir, searchString)

	wg.Wait()

	// The moment we close the channel, the main thread continues.
	// In the previous version, this resulted in a deadlock when no results
	// were found because it was using a blocking read from the channel.
	close(found)

	// To alleviate the issue noted above, we try to take out the lock
	// which is being used by the `collection` goroutine. As long as the
	// `collection` goroutine is running, our execution will block here
	// until the goroutine finishes and unlocks.
	resultsLock.Lock()

	// Now that the `collection` goroutine is done, we can do a non-blocking
	// read on the channel:

	select {
	// We needed to wait using the lock above because if we didn't wait,
	// this channel read would always come back as "empty" and trigger the default
	// thereby discarding results. This is because it takes the `collection`
	// goroutine time to gather the results into a slice.
	case hits := <-results:
		fmt.Println(hits)
	// If there are no results, then we just break and the program is done.
	default:
		break
	}

}
