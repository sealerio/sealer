/* Copyright (c) 2020 Bram Vandenbogaerde
 * You may use, distribute or modify this code under the
 * terms of the Mozilla Public License 2.0, which is distributed
 * along with the source code.
 */

package scp

import (
	"golang.org/x/crypto/ssh"
	"time"
)

// A struct containing all the configuration options
// used by an scp client.
type ClientConfigurer struct {
	host         string
	clientConfig *ssh.ClientConfig
	session      *ssh.Session
	timeout      time.Duration
	remoteBinary string
}

// Creates a new client configurer.
// It takes the required parameters: the host and the ssh.ClientConfig and
// returns a configurer populated with the default values for the optional
// parameters.
//
// These optional parameters can be set by using the methods provided on the
// ClientConfigurer struct.
func NewConfigurer(host string, config *ssh.ClientConfig) *ClientConfigurer {
	return &ClientConfigurer{
		host:         host,
		clientConfig: config,
		timeout:      0, // no timeout by default
		remoteBinary: "scp",
	}
}

// Sets the path of the location of the remote scp binary
// Defaults to: /usr/bin/scp
func (c *ClientConfigurer) RemoteBinary(path string) *ClientConfigurer {
	c.remoteBinary = path
	return c
}

// Alters the host of the client connects to
func (c *ClientConfigurer) Host(host string) *ClientConfigurer {
	c.host = host
	return c
}

// Changes the connection timeout.
// Defaults to one minute
func (c *ClientConfigurer) Timeout(timeout time.Duration) *ClientConfigurer {
	c.timeout = timeout
	return c
}

// Alters the ssh.ClientConfig
func (c *ClientConfigurer) ClientConfig(config *ssh.ClientConfig) *ClientConfigurer {
	c.clientConfig = config
	return c
}

// Alters the ssh.Session
func (c *ClientConfigurer) Session(session *ssh.Session) *ClientConfigurer {
	c.session = session
	return c
}

// Builds a client with the configuration stored within the ClientConfigurer
func (c *ClientConfigurer) Create() Client {
	return Client{
		Host:         c.host,
		ClientConfig: c.clientConfig,
		Timeout:      c.timeout,
		RemoteBinary: c.remoteBinary,
		Session:      c.session,
	}
}
