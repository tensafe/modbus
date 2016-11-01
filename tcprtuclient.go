// Copyright 2014 Quoc-Viet Nguyen. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license. See the LICENSE file for details.

package modbus

import (
	"io"
	"log"
	"net"
	"time"
)

// RTUClientHandler implements Packager and Transporter interface.
type TcpRTUClientHandler struct {
	rtuPackager
	tcpRtuTransporter
}

// NewRTUClientHandler allocates and initializes a RTUClientHandler.
func NewTcpRtuClientHandler(address string) *TcpRTUClientHandler {
	handler := &TcpRTUClientHandler{}
	handler.Address = address
	return handler
}

// tcpRtuTransporter implements Transporter interface.
type tcpRtuTransporter struct {
	// Connect string
	Address string
	// Connect & Read timeout
	Timeout time.Duration
	// Transmission logger
	Logger *log.Logger

	// TCP connection
	conn net.Conn
}

// Send sends data to server and ensures response length is greater than header length.
func (mb *tcpRtuTransporter) Send(aduRequest []byte) (aduResponse []byte, err error) {
	var data [tcpMaxLength]byte
	var n int
	var n1 int

	if mb.conn == nil {
		// Establish a new connection and close it when complete
		if err = mb.Connect(); err != nil {
			return
		}
		defer mb.Close()
	}
	if mb.Logger != nil {
		mb.Logger.Printf("modbus: sending % x\n", aduRequest)
	}
	if err = mb.conn.SetDeadline(time.Now().Add(mb.Timeout)); err != nil {
		return
	}
	if _, err = mb.conn.Write(aduRequest); err != nil {
		return
	}

	function := aduRequest[1]
	functionFail := aduRequest[1] & 0x80
	bytesToRead := calculateResponseLength(aduRequest)
	// Read header first
	if n, err = io.ReadAtLeast(mb.conn, data[:], rtuMinSize); err != nil {
		return
	}
	//if the function is correct
	if data[1] == function {
		//we read the rest of the bytes
		if n < bytesToRead {
			if bytesToRead > rtuMinSize && bytesToRead <= rtuMaxSize {
				if bytesToRead > n {
					n1, err = io.ReadFull(mb.conn, data[n:bytesToRead])
					n += n1
				}
			}
		}
	} else if data[1] == functionFail {
		//for error we need to read 5 bytes
		if n < bytesToRead {
			n1, err = io.ReadFull(mb.conn, data[n:5])
		}
		n += n1
	}

	if err != nil {
		return
	}
	aduResponse = data[:n]
	if mb.Logger != nil {
		mb.Logger.Printf("modbus: received % x\n", aduResponse)
	}

	return
}

// Connect establishes a new connection to the address in Address.
// Connect and Close are exported so that multiple requests can be done with one session
func (mb *tcpRtuTransporter) Connect() (err error) {
	// Timeout must be specified
	if mb.Timeout <= 0 {
		mb.Timeout = tcpTimeoutMillis * time.Millisecond
	}
	dialer := net.Dialer{Timeout: mb.Timeout}
	mb.conn, err = dialer.Dial("tcp", mb.Address)
	return
}

// Close closes current connection.
func (mb *tcpRtuTransporter) Close() (err error) {
	if mb.conn != nil {
		err = mb.conn.Close()
		mb.conn = nil
	}
	return
}

// flush flushes pending data in the connection,
// returns io.EOF if connection is closed.
func (mb *tcpRtuTransporter) flush(b []byte) (err error) {
	if err = mb.conn.SetReadDeadline(time.Now()); err != nil {
		return
	}
	// Timeout setting will be reset when reading
	if _, err = mb.conn.Read(b); err != nil {
		// Ignore timeout error
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			err = nil
		}
	}
	return
}
