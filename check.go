package goopy

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

// All the Check* functions in this file check if an error is nil and end the
// program gracefully if it isn't. This includes sending messages to the other
// processes.

// CheckChild checks that an error is non-nil and handles ending the program
// and sending appropriate signals if it isn't. CheckChild should only be
// called from a child process.
func CheckChild(pipe *Pipe, err error)  {
	if !pipe.IsChild {
		// This will ususally be caught by type-checking, since CheckParent and
		// CheckChild have different call signatures.
		err = fmt.Errorf("CheckChild called on a parent process.")
	}
	
	if err == nil { return }

	// Print a stack trce to the log file.
	stack := debug.Stack()
	log.Printf(string(stack) + "\n\n" + err.Error())
	
	pipe.encoder.Encode(errorCode)
	pipe.encoder.Encode(fmt.Sprintf(
		"Error occured in child process %d: %s", pipe.ID, err.Error()))
	
	os.Exit(1)
}

// CheckParent checks that an error is non-nil and handles ending the program
// and sending appropriate signals if it isn't. CheckChild should only be
// called from a child process.
func CheckParent(pipes []*Pipe, err error) {
	if len(pipes) > 0 && pipes[0].IsChild {
		// This will ususally be caught by type-checking, (CheckParent and
		// CheckChild have different call signatures ;-) )
		err = fmt.Errorf("CheckParent called on parent process.")
	}
	
	if err == nil { return }
	
	// Send an ending messages to each pipe and close them.
	for i := range pipes {
		pipes[i].Close()
	}

	// Append stack trace to error
	stack := debug.Stack()
	err = fmt.Errorf(`Symfind terminating due to error.

%s

Stack Trace:
%s"`, err.Error(), stack)

	// Send error to stderr and the log file.
	fmt.Fprintf(os.Stderr, err.Error())
	log.Fatal(err.Error())
}

// CheckPreSetup is an error check that can be done before the pipes have been
// fully set up.
func CheckPreSetup(err error) {
	if err == nil { return }
	
	// Append stack trace to error
	stack := debug.Stack()
	err = fmt.Errorf(`Symfind terminating due to error.

%s

Stack Trace:
%s"`, err.Error(), stack)

	// Send error to stderr and the log file.
	fmt.Fprintf(os.Stderr, err.Error())
	log.Fatal(err.Error())
}
