package goopy

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"	
)

type ParentConfig struct {
	Plugin string
	IPCBase string
	LogBase string
	PluginWorkers int64
	PID int64
}

type ChildConfig struct {
	ID int64
	PPID int64
	IPCBase string
	LogBase string
}

// SetupParent starts Symfind on a parent process. If cfg is nil, the command
// line is parsed to get the config. SetupParent returns pipes to every child
// process and the config file.
func SetupParent(cfg *ParentConfig) ([]*Pipe, *Runtime) {
	var err error
	if cfg == nil {
		cfg = parseParentArgs()
		CheckPreSetup(err)
	}
	cfg.PID = int64(os.Getpid())

	
	f, err := setLogFile(cfg.LogBase + ".parent")
	CheckPreSetup(err)
	
	pipes := make([]*Pipe, cfg.PluginWorkers)
	for i := range pipes {
		command := createPluginCommand(cfg, i)

		base := fmt.Sprintf("%s.%d", cfg.IPCBase, i)
		pipes[i], err = StartProcess(base, command...)
		CheckPreSetup(err)
		pipes[i].ID = int64(i)

		monitorChild(pipes, i)
	}

	return pipes, NewRuntime(f, cfg.IPCBase)
}

// SetupChild starts Symfind on a child process. If cfg is nil, the command
// line is parsed to get the config. SetupParent returns pipes to every child
// process and the config file.
func SetupChild(cfg *ChildConfig) (*Pipe, *Runtime) {
	var err error
	if cfg == nil {
		cfg, err = parseChildArgs()
		CheckPreSetup(err)
	}

	f, err := setLogFile(fmt.Sprintf("%s.child.%d", cfg.LogBase, cfg.ID))
	CheckPreSetup(err)
	
	pipe, err := Listen(cfg.IPCBase, cfg.ID)
	CheckPreSetup(err)

	monitorParent(pipe, cfg.PPID)
	
	return pipe, NewRuntime(f, cfg.IPCBase)
}

// createPluginCommand generates a command to start running 
func createPluginCommand(cfg *ParentConfig, id int) []string {
	var command []string

	// If the plugin is a Go or Python file, run the source file directly,
	// otherwise assume its a binary.
	if i := strings.LastIndex(cfg.Plugin, ".go"); i != -1 {
		command = []string{ "go", "run", cfg.Plugin }
	} else if i := strings.LastIndex(cfg.Plugin, ".py"); i != -1 {
		// Some people have different aliases for Python 3 and Python 2, so
		// check for that first.
		if _, err := exec.LookPath("python3"); err != nil {
			command = []string{ "python3", cfg.Plugin }
		} else {
			command = []string{ "python", cfg.Plugin }
		}
	} else {
		command = []string{ cfg.Plugin }
	}

	// Add flags to the base 
	command = append(command, fmt.Sprintf("-id=%d", id))
	command = append(command, fmt.Sprintf("-ipc-base=%s.%d", cfg.IPCBase, id))
	command = append(command, fmt.Sprintf("-log-base=%s", cfg.LogBase))
	command = append(command, fmt.Sprintf("-ppid=%d", cfg.PID))

	return command
}

func monitorChild(pipes []*Pipe, idx int) {
	// Set up a monitor for the child process.
	ticker := time.Tick(time.Second)
	go func() {
		for {
			<-ticker
			if !pipes[idx].IsRunning() {
				CheckParent(pipes, fmt.Errorf("Child process %d has ended " +
					"without notifying the parent.", idx))
			}
		}
	}()
}

func monitorParent(pipe *Pipe, pid int64) {
	ticker := time.Tick(time.Second)
	go func() {
		for {
			<- ticker
			if _, err := os.FindProcess(int(pid)); err != nil {
				CheckChild(pipe, fmt.Errorf("Parent proces has ended " +
					"without notifying the child"))
			}
		}
	}()
}

// setLogFile sets log output to go to a new file with the given name. The file
// handler is returned.
func setLogFile(fname string) (*os.File, error) {
	f, err := os.Create(fname)
	if err != nil { return nil, err }
	
	log.SetOutput(f)

	return f, nil
}

// parseParentArgs parses and validates command line arguments for a parent
// process.
func parseParentArgs() (*ParentConfig) {
	args := &ParentConfig{ }

	set := &flag.FlagSet{ }
	
	// Define parsers.
	set.StringVar(&args.Plugin, "plugin", "",
		"The plugin file to run. (Must be .py, .go, or a binary file.)")
	set.StringVar(&args.IPCBase, "ipc-base", "test_files/ipc",
		"The base of the named unix pipes you want to create.")
	set.StringVar(&args.LogBase, "log-base", "test_files/log",
		"The base of the log files")
	set.Int64Var(&args.PluginWorkers, "plugin-workers", 1,
		"The number of worker processes to spawn.")

	// Parse.
	err := set.Parse(os.Args[1:])
	if err != nil {
		// It's too early in the program to even have a log file yet.
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Check for valid input.
	if args.Plugin == "" {
		err = fmt.Errorf("Must set -plugin")
	} else if args.IPCBase == "" {
		err = fmt.Errorf("Must set -ipc-base")
	} else if args.PluginWorkers < 1 {
		err = fmt.Errorf("-plugin-workers must be set to a positive number")
	} else if args.PluginWorkers > 1<<10 {
		err = fmt.Errorf("-plugin-workers must be less than %d (also, " +
			"don't set it that high)", 1<<10)
	}

	// Also too early to send the errors to the log file.
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error() +
			"(run with -help to see flags).")
		os.Exit(1)
	}
	
	return args
}

// parseChildArgs parses and validates command line arguments for a child
// process.
func parseChildArgs() (*ChildConfig, error) {
	args := &ChildConfig{ }

	// Define parsers.
	flag.Int64Var(&args.ID, "id", -1,
		"Unique ID of the child process.")
	flag.StringVar(&args.IPCBase, "ipc-base", "test_files/ipc",
		"The base of the named unix pipes you want to create.")
	flag.StringVar(&args.LogBase, "log-base", "test_files/log",
		"The base of the log files.")
	flag.Int64Var(&args.PPID, "ppid", -1,
		"PID of the parent process spawning the child/")

	// Parse.
	flag.Parse()

	// Check for valid input.
	if args.IPCBase == "" {
		return nil, fmt.Errorf("Must set --ipc-base")
	} else if args.ID == -1 {
		return nil, fmt.Errorf("Must set --id")
	} else if  args.PPID == -1 {
		return nil, fmt.Errorf("Must set --ppid")
	}
	
	return args, nil
}
