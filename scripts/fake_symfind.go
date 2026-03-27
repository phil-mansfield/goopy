package main

import (
	"log"
	"os"
	
	"github.com/phil-mansfield/goopy"
)

func main() {
	err := ResetDir("test_files")
	goopy.CheckPreSetup(err)
		
	pipes, r := goopy.SetupParent(nil)
	_, _ = pipes, r

	log.Println("\nStarted Parent process")
	log.Printf("Arguments: %s", os.Args)

	for i := range pipes {
		data := make([]float32, 10)
		for j := range data { data[j] = float32(i) }
		log.Println("Parent about to SendData")
		err = goopy.SendData(pipes[i], data)
		log.Println("Parent finished SendData with error", err)
		goopy.CheckParent(pipes, err)


		log.Println("Parent about to RecvMsg")
		var msg string
		err = goopy.RecvMsg(pipes[i], &msg)
		log.Println("Parent finished RecvMsg with error", err)
		goopy.CheckParent(pipes, err)
		
		log.Println("Parent recieved the message:", msg)
		
	}

	for i := range pipes { pipes[i].Close() }
}

// ResetDir sets the given directory to an empty state (i.e. creates it if
// it doesn't exist and removes everything in it if it does exist).
func ResetDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		// Create directory if it doesn't exist
		err := os.Mkdir(dir, 0750)
		if err != nil { panic(err.Error()) }
	} else {
		// Remove all the files in the directory otherwise.
		entries, err := os.ReadDir(dir)
		if err != nil { return err }
		
		for _, entry := range entries {
			if !entry.IsDir() {
				err := os.Remove(dir + "/" + entry.Name())
				if err != nil { return err }
			}
		}
	}
	
	return nil
}
