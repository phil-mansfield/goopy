package main

import (
	"log"
	"os"
	"flag"
	"fmt"
	"slices"
	
	"github.com/phil-mansfield/goopy"
)

func main() {
	role, mode := ParseArgs()
	if role == "luna" {
		ResetDir("test_files")
	}
	
	// Set up log files
	f, err := os.Create("test_files/" + role + ".log")
	if err != nil { panic(err.Error()) }
	defer f.Close()
	log.SetOutput(f)
	
	log.Println(os.Args)
	
	suggestedMode := goopy.SuggestMode()
	log.Println("Goopy suggest mode:", suggestedMode)
	
	if !slices.Contains([]string{"apipe", "npipe", "mmap", "socket"}, mode) {
		panic(fmt.Sprintf("Unrecognised mode, '%s'", mode))
	}

	switch role {
	case "luna":
		Luna(mode)
	case "mouse":
		Mouse(mode)
	}
}

func ParseArgs() (role, mode string) {	
	flag.StringVar(&role, "role", "",
		"The role that the test binary is running in. [luna | mouse]")
	flag.StringVar(&mode, "mode", "",
		"Communication method [apipe | npipe | mmap | socket]")

	flag.Parse()

	return
}

func Luna(mode string) {	
	pipe, err := goopy.StartProcess(mode, "test_files/ipc",
		"./bin/goopy", "--mode="+mode, "--role=mouse")
	if err != nil { panic(err.Error()) }

	log.Println("Luna says: Meow!")

	goopy.SendMssg(pipe, "Meow! I'm so scared of you!")
	if err != nil { panic(err.Error()) }

	buf := []int32{ }
	err = goopy.RecvData(pipe, &buf)
	if err != nil { log.Panic(err.Error()) }

	log.Println("Lura recieved", buf)
	
	var done string
	err = goopy.RecvMssg(pipe, &done)
	if err != nil { panic(err.Error()) }
}

func Mouse(mode string) {
	pipe, err := goopy.Listen(mode, "test_files/ipc")
	if err != nil { log.Panic(err.Error()) }

	log.Println("Mouse says: Squeak!")

	var firstMssg string
	err = goopy.RecvMssg(pipe, &firstMssg)
	if err != nil { log.Panic(err.Error()) }

	log.Println("The mouse hears:", firstMssg)
	log.Println("The mouse recites math to Luna")
	
	err = goopy.SendData(pipe, []int32{1, 1, 2, 3, 5, 8, 13})
	if err != nil { log.Panic(err.Error()) }
	
	err = goopy.SendMssg(pipe, "done")
	if err != nil { log.Panic(err.Error()) }
}

func ResetDir(dir string) {
	if _, err := os.Stat(dir); err != nil {
		// Create directory if it doesn't exist
		err := os.Mkdir(dir, 0750)
		if err != nil { panic(err.Error()) }
	} else {
		// Remove all the files in the directory otherwise.
		entries, err := os.ReadDir(dir)
		if err != nil { panic(err.Error()) }
		
		for _, entry := range entries {
			if !entry.IsDir() {
				err := os.Remove(dir + "/" + entry.Name())
				if err != nil { panic(err.Error()) }
			}
		}
	}
}
