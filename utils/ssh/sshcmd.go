package ssh

import (
	"bufio"
	"fmt"
	"io"

	"github.com/alibaba/sealer/logger"
)

func (s *SSH) Ping(host string) error {
	_, err := s.Connect(host)
	if err != nil {
		return fmt.Errorf("[ssh %s]create ssh session failed, %v", host, err)
	}
	return nil
}

func (s *SSH) CmdAsync(host string, cmds ...string) error {
	var flag bool

	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		func(cmd string) {
			session, err := s.Connect(host)
			if err != nil {
				flag = true
				logger.Error("[ssh %s]create ssh session failed, %s", host, err)
				return
			}
			defer session.Close()
			logger.Info("[ssh][%s] : %s", host, cmd)
			stdout, err := session.StdoutPipe()
			if err != nil {
				flag = true
				logger.Error("[ssh %s]create stdout pipe failed, %s", host, err)
				return
			}
			stderr, err := session.StderrPipe()
			if err != nil {
				flag = true
				logger.Error("[ssh %s]create stderr pipe failed, %s", host, err)
				return
			}
			if err := session.Start(cmd); err != nil {
				flag = true
				logger.Error("[%s]run command failed, %v", cmd, err)
				return
			}
			doneout := make(chan bool, 1)
			doneerr := make(chan bool, 1)
			go func() {
				readPipe(stderr, true)
				doneerr <- true
			}()
			go func() {
				readPipe(stdout, false)
				doneout <- true
			}()
			<-doneerr
			<-doneout
			err = session.Wait()
			if err != nil {
				flag = true
				logger.Error("exec command failed %v", err)
				return
			}
		}(cmd)
		if flag {
			return fmt.Errorf("exec command failed %s %s", host, cmd)
		}
	}

	return nil
}

func (s *SSH) Cmd(host, cmd string) ([]byte, error) {
	//logger.Info("[ssh][%s] %s", host, cmd)
	session, err := s.Connect(host)
	if err != nil {
		return nil, fmt.Errorf("[ssh][%s] create ssh session failed, %s", host, err)
	}
	defer session.Close()
	b, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("[ssh][%s]failed to run command [%s],output is: %s", host, cmd, b)
		return nil, fmt.Errorf("[ssh][%s]run command failed [%s]", host, cmd)
	}
	return b, nil
}

func readPipe(pipe io.Reader, isErr bool) {
	r := bufio.NewReader(pipe)
	for {
		line, _, err := r.ReadLine()
		if line == nil {
			return
		}
		// should not using logger
		fmt.Println(string(line))
		if err != nil {
			logger.Error("%v", err)
			return
		}
	}
}
