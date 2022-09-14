/* Copyright (c) 2021 Bram Vandenbogaerde And Contributors
 * You may use, distribute or modify this code under the
 * terms of the Mozilla Public License 2.0, which is distributed
 * along with the source code.
 */
package scp

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

type ResponseType = uint8

const (
	Ok      ResponseType = 0
	Warning ResponseType = 1
	Error   ResponseType = 2
)

const buffSize = 1024 * 256

// There are tree types of responses that the remote can send back:
// ok, warning and error
//
// The difference between warning and error is that the connection is not closed by the remote,
// however, a warning can indicate a file transfer failure (such as invalid destination directory)
// and such be handled as such.
//
// All responses except for the `Ok` type always have a message (although these can be empty)
//
// The remote sends a confirmation after every SCP command, because a failure can occur after every
// command, the response should be read and checked after sending them.
type Response struct {
	Type    ResponseType
	Message string
}

// Reads from the given reader (assuming it is the output of the remote) and parses it into a Response structure
func ParseResponse(reader io.Reader) (Response, error) {
	buffer := make([]uint8, 1)
	_, err := reader.Read(buffer)
	if err != nil {
		return Response{}, err
	}

	response_type := buffer[0]
	message := ""
	if response_type > 0 {
		buffered_reader := bufio.NewReader(reader)
		message, err = buffered_reader.ReadString('\n')
		if err != nil {
			return Response{}, err
		}
	}

	return Response{response_type, message}, nil
}

func (r *Response) IsOk() bool {
	return r.Type == Ok
}

func (r *Response) IsWarning() bool {
	return r.Type == Warning
}

// Returns true when the remote responded with an error
func (r *Response) IsError() bool {
	return r.Type == Error
}

// Returns true when the remote answered with a warning or an error
func (r *Response) IsFailure() bool {
	return r.Type > 0
}

// Returns the message the remote sent back
func (r *Response) GetMessage() string {
	return r.Message
}

type FileInfos struct {
	Message     string
	Filename    string
	Permissions string
	Size        int64
}

func (r *Response) ParseFileInfos() (*FileInfos, error) {
	message := strings.ReplaceAll(r.Message, "\n", "")
	parts := strings.Split(message, " ")
	if len(parts) < 3 {
		return nil, errors.New("Unable to parse message as file infos")
	}

	size, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	return &FileInfos{
		Message:     r.Message,
		Permissions: parts[0],
		Size:        int64(size),
		Filename:    parts[2],
	}, nil
}

// Writes an `Ack` message to the remote, does not await its response, a seperate call to ParseResponse is
// therefore required to check if the acknowledgement succeeded
func Ack(writer io.Writer) error {
	var msg = []byte{0}
	n, err := writer.Write(msg)
	if err != nil {
		return err
	}
	if n < len(msg) {
		return errors.New("Failed to write ack buffer")
	}
	return nil
}
