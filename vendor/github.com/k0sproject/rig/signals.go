//go:build !windows
// +build !windows

package rig

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	ssh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func captureSignals(stdin io.WriteCloser, session *ssh.Session) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTSTP, syscall.SIGWINCH)

	go func() {
		for sig := range ch {
			switch sig {
			case os.Interrupt:
				fmt.Fprintf(stdin, "\x03")
			case syscall.SIGTSTP:
				fmt.Fprintf(stdin, "\x1a")
			case syscall.SIGWINCH:
				_, err := session.SendRequest("window-change", false, termSizeWNCH())
				if err != nil {
					println("failed to relay window-change event: " + err.Error())
				}
			}
		}
	}()
}

func termSizeWNCH() []byte {
	size := make([]byte, 16)
	fd := int(os.Stdin.Fd())
	rows, cols, err := term.GetSize(fd)
	if err != nil {
		binary.BigEndian.PutUint32(size, 40)
		binary.BigEndian.PutUint32(size[4:], 80)
	} else {
		binary.BigEndian.PutUint32(size, uint32(cols))
		binary.BigEndian.PutUint32(size[4:], uint32(rows))
	}

	return size
}
