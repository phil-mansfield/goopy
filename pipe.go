package goopy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"slices"
	
	"encoding/binary"
	"encoding/gob"
)

// Pipe represents a pipe between a child and parent process
type Pipe struct {
	ID int64
	
	Cmd *exec.Cmd
	IsChild bool
	BaseFile string
	
	GobReader io.ReadCloser
	GobWriter io.WriteCloser
	BinaryReader io.ReadCloser
	BinaryWriter io.WriteCloser
	
	decoder *gob.Decoder
	encoder *gob.Encoder
}

// IsRunning returns true if the process on the other end of the pipe is still
// running. Children are automatically ended when the parent ends, 
func (p *Pipe) IsRunning() bool {
	if p.IsChild { return true }
	
	// Signal 0 is a no-op, so the error only checks is the process is still
	// running.
	err := p.Cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Close ends the underlying process and closes its unix fifos.
func (pipe *Pipe) Close() {
	// Parent processes should also kill the child processes.
	if !pipe.IsChild { pipe.Cmd.Process.Kill() }
	
	pipe.GobWriter.Close()
	pipe.GobReader.Close()
	pipe.BinaryWriter.Close()
	pipe.BinaryReader.Close()
	os.Remove(pipe.BaseFile + ".0")
	os.Remove(pipe.BaseFile + ".1")
	os.Remove(pipe.BaseFile + ".2")
	os.Remove(pipe.BaseFile + ".3")
}


// StartProcess starts a child subprocess corresponding to the given command.
// baseName is the base file name for the unix pipes used for interprocess
// communication.
func StartProcess(baseFile string, command ...string) (*Pipe, error) {
	if len(command) < 1 {
		return nil, fmt.Errorf("Must supply a command to StartProcess")
	}
	
	var err error	
	
	pipe := &Pipe{ }
	pipe.Cmd = exec.Command(command[0], command[1:]...)
	pipe.BaseFile = baseFile
		
	// 1: child -> parent
	// 2: parent -> child
	pipe0, pipe1 := baseFile + ".0", baseFile + ".1"
	pipe2, pipe3 := baseFile + ".2", baseFile + ".3"
	
	// mode 0640: user read/write permissions, group read permissions
	syscall.Mkfifo(pipe0, 0640)
	syscall.Mkfifo(pipe1, 0640)
	syscall.Mkfifo(pipe2, 0640)
	syscall.Mkfifo(pipe3, 0640)
	
	// Start the command after creating the pipes to avoid errors in the child
	// processes.
	err = pipe.Cmd.Start()
	if err != nil { return nil, err }
	
	// TODO: This is going to be a very common error if, e.g., the child
	// process isn't really a plugin file or messes up the boilerplate. We need
	// to make this error message more user-firendly (the user shouldn't know
	// anything about the flags).
	if !pipe.IsRunning() {
		CheckPreSetup(fmt.Errorf(`A child process terminated before connecting with the parent process. The child process was generated with the following command:
%s

This is a very hard error to give you a useful error message on. It is usually caused by passing something that is not a valid plugin to symfind, but was occasionlly caused by internal errors during early development.

Make sure to also check the corresponding child log file if it exists.`, command))
	}

	pipe.GobReader, err = os.OpenFile(pipe0, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.GobWriter, err = os.OpenFile(pipe1, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.BinaryReader, err = os.OpenFile(pipe2, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.BinaryWriter, err = os.OpenFile(pipe3, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	// Make gob encoders for messages.
	pipe.encoder = gob.NewEncoder(pipe.GobWriter)
	pipe.decoder = gob.NewDecoder(pipe.GobReader)

	return pipe, nil
}

func Listen(baseFile string, id int64) (*Pipe, error) {
	// Communication is done with a pair of unix fifo named pipes. 
	// 1: child -> parent
	// 2: parent -> child
	
	pipe := &Pipe{ }

	pipe.IsChild = true
	pipe.ID = id
	pipe.BaseFile = baseFile
	
	pipe0, pipe1 := baseFile + ".0", baseFile + ".1"
	pipe2, pipe3 := baseFile + ".2", baseFile + ".3"
	
	// Open the pipes
	var err error
	pipe.GobWriter, err = os.OpenFile(pipe0, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.GobReader, err = os.OpenFile(pipe1, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.BinaryWriter, err = os.OpenFile(pipe2, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	pipe.BinaryReader, err = os.OpenFile(pipe3, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil { return nil, err }

	// Make gob encoders for messages.
	pipe.encoder = gob.NewEncoder(pipe.GobWriter)
	pipe.decoder = gob.NewDecoder(pipe.GobReader)
	
	return pipe, nil
}

// Type codes are sent at the beginning of the message 
type typeCode byte
const (
	errorCode typeCode = iota
	messageCode
	dataCode
)

// SendMsg sends a "message" to the other end of the pipe. Messages can be any
// type. Messages use a slower and more generic encoding method which may lead
// to heap allocations.
func SendMsg(p *Pipe, v any) error {
	err := p.encoder.Encode(messageCode)
	err = p.checkInterruption(err)
	if err != nil { return err }
	
	// Send message.
	err = p.encoder.Encode(v)
	return p.checkInterruption(err)
}

func SendData[S ~[]E, E any](p *Pipe, s S) error {
	err := p.encoder.Encode(dataCode)
	err = p.checkInterruption(err)
	if err != nil { return err }
	
	// Send size of array.
	err = binary.Write(p.BinaryWriter, binary.LittleEndian, uint64(len(s)))
	p.checkInterruption(err)
	if err != nil { return err }
	
	// Send array
	err = binary.Write(p.BinaryWriter, binary.LittleEndian, s)
	return p.checkInterruption(err)
}

// RecvMsg recieves a "message" to the other end of the pipe. Messages can be
// any type. Messages use a slower and more generic encoding method which may
// lead to heap allocations.
func RecvMsg(p *Pipe, v any) error {	
	// The error checking here isn't very pretty.
	var flag typeCode
	err := p.decoder.Decode(&flag)
	err = p.checkInterruption(err)
	if err != nil { return err }

	// Check if an unexpected message has been sent.
	if flag == errorCode {
		// The other side of the pipe has reported an error.
		var errorText string
		err = p.decoder.Decode(&errorText)
		err = p.checkInterruption(err)
		if err != nil { return err }
		
		return fmt.Errorf(errorText)
	} else if flag == dataCode {
		// User error with a mismatch between message and data.
		return fmt.Errorf("Code is expecting to recieve a message but " +
			"recieved data instead.")
	}

	// Finally, decode the message.
	err = p.decoder.Decode(v)
	return p.checkInterruption(err)
}

func RecvData[S ~[]E, E any](p *Pipe, s *S) error {
	// The error checking here isn't very pretty.
	
	// Read typeCode byte first
	var flag typeCode
	err := p.decoder.Decode(&flag)
	err = p.checkInterruption(err)
	if err != nil { return err }
	
	// Check if an unexpected message has been sent
	if flag == errorCode {
		// The other side of the pipe has reported an error.
		var errorText string
		err = p.decoder.Decode(&errorText)
		err = p.checkInterruption(err)
		if err != nil { return err }
		
		return fmt.Errorf(errorText)
	} else if flag == messageCode {
		return fmt.Errorf("Code is expecting to recieve data, but recieved " +
			"a message instead.")
	}
	
	// Read array size.
	n := uint64(0)
	err = binary.Read(p.BinaryReader, binary.LittleEndian, &n)
	err = p.checkInterruption(err)
	if err != nil { return err }

	// Resize buffer to the correct size and read into it.
	*s = slices.Grow(*s, int(n))[:n]
	err = binary.Read(p.BinaryReader, binary.LittleEndian, s)
	return p.checkInterruption(err)
}

func (p *Pipe) checkInterruption(err error) error {
	if err == nil { return nil }
	if err.Error() != "EOF" { return err }
	if p.IsChild {
		return fmt.Errorf("Parent process ended unexpectedly without " +
			"notifying child process %d.", p.ID)
	} else {
		return fmt.Errorf("Child process %d ended unexpectedly without " +
			"notifying parent process.", p.ID)
	}
}
