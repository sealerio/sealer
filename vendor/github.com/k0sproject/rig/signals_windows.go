// +build windows

package rig

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	ssh "golang.org/x/crypto/ssh"
)

func captureSignals(stdin io.WriteCloser, session *ssh.Session) {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)

	go func() {
		for sig := range ch {
			switch sig {
			case os.Interrupt:
				fmt.Fprintf(stdin, "\x03")
			}
		}
	}()
}
