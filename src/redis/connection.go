//   Copyright 2009-2012 Joubin Houshyar
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package redis

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// connection socket modes
const (
	TCP  = "tcp"  // tcp/ip socket
	UNIX = "unix" // unix domain socket
)

// various defaults for the connections
// exported for user convenience.
const (
	DefaultReqChanSize          = 1000000
	DefaultRespChanSize         = 1000000
	DefaultTCPReadBuffSize      = 1024 * 256
	DefaultTCPWriteBuffSize     = 1024 * 256
	DefaultTCPReadTimeoutNSecs  = 1000 * time.Nanosecond
	DefaultTCPWriteTimeoutNSecs = 1000 * time.Nanosecond
	DefaultTCPLinger            = 0 // -n: finish io; 0: discard, +n: wait for n secs to finish
	DefaultTCPKeepalive         = true
	DefaultHeartbeatSecs        = 1 * time.Second
	DefaultProtocol             = REDIS_DB
)

// Redis specific default settings
// exported for user convenience.
const (
	DefaultRedisPassword = ""
	DefaultRedisDB       = 0
	DefaultRedisPort     = 6379
	DefaultRedisHost     = "127.0.0.1"
)

// ----------------------------------------------------------------------------
// Connection ConnectionSpec
// ----------------------------------------------------------------------------

type Protocol int

const (
	REDIS_DB Protocol = iota
	REDIS_PUBSUB
)

func (p Protocol) String() string {
	switch p {
	case REDIS_DB:
		return "Protocol:REDIS_DB"
	case REDIS_PUBSUB:
		return "Protocol:PubSub"
	}
	return "BUG - unknown protocol value"
}

// Defines the set of parameters that are used by the client connections
//
type ConnectionSpec struct {
	host       string        // redis connection host
	port       int           // redis connection port
	password   string        // redis connection password
	db         int           // Redis connection db #
	rBufSize   int           // tcp read buffer size
	wBufSize   int           // tcp write buffer size
	rTimeout   time.Duration // tcp read timeout
	wTimeout   time.Duration // tcp write timeout
	keepalive  bool          // keepalive flag
	lingerspec int           // -n: finish io; 0: discard, +n: wait for n secs to finish
	reqChanCap int           // async request channel capacity - see DefaultReqChanSize
	rspChanCap int           // async response channel capacity - see DefaultRespChanSize
	heartbeat  time.Duration // 0 means no heartbeat
	protocol   Protocol
}

// Creates a ConnectionSpec using default settings.
// using the DefaultXXX consts of redis package.
func DefaultSpec() *ConnectionSpec {
	return &ConnectionSpec{
		DefaultRedisHost,
		DefaultRedisPort,
		DefaultRedisPassword,
		DefaultRedisDB,
		DefaultTCPReadBuffSize,
		DefaultTCPWriteBuffSize,
		DefaultTCPReadTimeoutNSecs,
		DefaultTCPWriteTimeoutNSecs,
		DefaultTCPKeepalive,
		DefaultTCPLinger,
		DefaultReqChanSize,
		DefaultRespChanSize,
		DefaultHeartbeatSecs,
		DefaultProtocol,
	}
}

// Sets the db for connection spec and returns the reference
// Note that you should not this after you have already connected.
func (spec *ConnectionSpec) Db(db int) *ConnectionSpec {
	spec.db = db
	return spec
}

// Sets the host for connection spec and returns the reference
// Note that you should not this after you have already connected.
func (spec *ConnectionSpec) Host(host string) *ConnectionSpec {
	spec.host = host
	return spec
}

// Sets the port for connection spec and returns the reference
// Note that you should not this after you have already connected.
func (spec *ConnectionSpec) Port(port int) *ConnectionSpec {
	spec.port = port
	return spec
}

// Sets the password for connection spec and returns the reference
// Note that you should not this after you have already connected.
func (spec *ConnectionSpec) Password(password string) *ConnectionSpec {
	spec.password = password
	return spec
}

// return the address as string.
func (spec *ConnectionSpec) Heartbeat(period time.Duration) *ConnectionSpec {
	spec.heartbeat = period
	return spec
}

// return the address as string.
func (spec *ConnectionSpec) Protocol(protocol Protocol) *ConnectionSpec {
	spec.protocol = protocol
	return spec
}

// ----------------------------------------------------------------------------
// SyncConnection API
// ----------------------------------------------------------------------------

// Defines the service contract supported by synchronous (Request/Reply)
// connections.

type SyncConnection interface {
	ServiceRequest(cmd *Command, argv []string) (Response, Error)
}

// ----------------------------------------------------------------------------
// Generic Conn handle and methods - supports SyncConnection interface
// ----------------------------------------------------------------------------

// General control structure used by connections.
//
type connHdl struct {
	spec      *ConnectionSpec
	conn      net.Conn // may want to change this to TCPConn - TODO REVU
	reader    *bufio.Reader
	connected bool // TODO
}

// Returns minimal info string for logging, etc
func (c *connHdl) String() string {
	return fmt.Sprintf("conn<redis-server@%s:%d [db %d]>", c.spec.host, c.spec.port, c.spec.db)
}

// Creates and opens a new connection to server per ConnectionSpec.
// The new connection is wrapped by a new connHdl with its bufio.Reader
// delegating to the net.Conn's reader.
//
// panics on error (with error)
func newConnHdl(spec *ConnectionSpec) (hdl *connHdl) {
	loginfo := "newConnHdl"

	hdl = new(connHdl)
	// REVU - this is silly
	if hdl == nil {
		panic(fmt.Errorf("%s(): failed to allocate connHdl", loginfo))
	}

	var mode, addr string
	if spec.port == 0 { // REVU - no special values (it was a contrib) TODO add flag to connspec.
		mode = UNIX
		addr = spec.host
	} else {
		mode = TCP
		addr = fmt.Sprintf("%s:%d", spec.host, spec.port)
		_, e := net.ResolveTCPAddr(TCP, addr)
		if e != nil {
			panic(fmt.Errorf("%s(): failed to resolve remote address %s", loginfo, addr))
		}
	}

	conn, e := net.Dial(mode, addr)
	switch {
	case e != nil:
		panic(fmt.Errorf("%s(): could not open connection", loginfo))
	case conn == nil:
		panic(fmt.Errorf("%s(): net.Dial returned nil, nil (?)", loginfo))
	default:
		configureConn(conn, spec)
		hdl.spec = spec
		hdl.conn = conn
		hdl.connected = true
		bufsize := 4096
		hdl.reader = bufio.NewReaderSize(conn, bufsize)
	}
	return
}

func configureConn(conn net.Conn, spec *ConnectionSpec) {
	// REVU - this requires a refact of protocol.go's error propagation
	// starting from read or write op -- 09-22-2012
	// TODO 09-23-2012
	// Deadline can be set in a handful of callsites in connection.go
	// but need a clean way to test for net.Error (as cause) and timeout
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetLinger(spec.lingerspec)
		tcp.SetKeepAlive(spec.keepalive)
		tcp.SetReadBuffer(spec.rBufSize)
		tcp.SetWriteBuffer(spec.wBufSize)
	}
}

// connect event handler will issue AUTH/SELECT on new connection
// if required.
// panics on error (with error)
func (c *connHdl) connect() {
	if c.spec.password != DefaultRedisPassword {
		args := []string{ c.spec.password }
		if _, e := c.ServiceRequest(&AUTH, args); e != nil {
			panic(e)
			//			panic(fmt.Errorf("<ERROR> Authentication failed - %s", e.Message()))
		}
	}
	if c.spec.db != DefaultRedisDB {
		args := []string{fmt.Sprintf("%d", c.spec.db)}
		if _, e := c.ServiceRequest(&SELECT, args); e != nil {
			panic(e)
			//			panic(fmt.Errorf("<ERROR> REDIS_DB Select failed - %s", e.Message()))
		}
	}
	// REVU - pretty please TODO do the customized log
	//	log.Printf("<INFO> %s - CONNECTED", c)
	return
}

// disconnects from net connections and sets connected state to false
// panics on net error (with error)
func (hdl *connHdl) disconnect() {
	// silently ignore repeated calls to closed connections
	if hdl.connected {
		if e := hdl.conn.Close(); e != nil {
			panic(fmt.Errorf("on connHdl.Close()", e))
			//			return newSystemErrorWithCause( "on connHdl.Close()", e)
		}
		hdl.connected = false
		// REVU - pretty please TODO do the customized log
		//		log.Printf("<INFO> %s - DISCONNECTED", hdl)
	}
}

// Creates a new SyncConnection using the provided ConnectionSpec.
// Note that this function will also connect to the specified redis server.
func NewSyncConnection(spec *ConnectionSpec) (c SyncConnection, err Error) {
	defer func() {
		if e := recover(); e != nil {
			connerr := e.(error)
			err = newSystemErrorWithCause("NewSyncConnection", connerr)
		}
	}()

	connHdl := newConnHdl(spec)
	connHdl.connect()
	c = connHdl
	return
}

// Implementation of SyncConnection.ServiceRequest.
func (c *connHdl) ServiceRequest(cmd *Command, argv []string) (resp Response, err Error) {
	args := make([][]byte, len(argv), len(argv))
	for i := range argv {
		args[i] = []byte(argv[i])
	}
	loginfo := "connHdl.ServiceRequest"

	defer func() {
		if re := recover(); re != nil {
			// REVU - needs to be logged - TODO
			err = newSystemErrorWithCause("ServiceRequest", re.(error))
		}
	}()

	if !c.connected {
		panic(fmt.Errorf("Connection %s is alredy closed", c.String()))
	}

	if cmd == &QUIT {
		c.disconnect()
		return
	}

	buff := CreateRequestBytes(cmd, args)
	sendRequest(c.conn, buff) // panics

	// REVU - this demands resp to be non-nil even in case of io errors
	// TODO - look into this
	resp, e := GetResponse(c.reader, cmd)
	if e != nil {
		panic(fmt.Errorf("%s(%s) - failed to get response", loginfo, cmd.Code))
	}

	// handle Redis server ERR - don't panic
	if resp.IsError() {
		redismsg := fmt.Sprintf(" [%s]: %s", cmd.Code, resp.GetMessage())
		err = newRedisError(redismsg)
	}

	return
}
