package rcon

// Minimal client for the Source RCON protocol (used by CS2).
// Wire format per packet: int32 size, int32 id, int32 type, body, two null bytes.
//
// CS2 gives no end-of-response marker, so a response is read until the server
// goes quiet. Two things make that safe:
//
//   - Responses echo the id of the command that caused them, so output still
//     arriving from an earlier command is recognised and dropped instead of
//     being returned as this command's answer.
//   - Waiting for a response to *start* and reading a packet that has already
//     started are bounded separately. A packet whose header is in the socket
//     is on its way, so its body is never cut off by the short between-packets
//     wait, however slow the link.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	packetExec         = 2 // SERVERDATA_EXECCOMMAND
	packetAuth         = 3 // SERVERDATA_AUTH
	packetResponse     = 0 // SERVERDATA_RESPONSE_VALUE
	packetAuthResponse = 2 // SERVERDATA_AUTH_RESPONSE

	maxPacketSize = 1 << 20
)

const (
	// How long a server may take to begin answering. `maps *` on a busy
	// server is far slower than a status poll, and a connection over the
	// internet adds to it. This used to be 2s, which quietly turned a slow
	// answer into an empty one.
	responseStartWait = 8 * time.Second
	// Once an answer has started, how long to wait for more of it before
	// treating it as complete.
	responseQuietWait = 400 * time.Millisecond
	// Budget for the body of a packet whose header has already arrived.
	packetBodyWait = 15 * time.Second

	dialTimeout = 5 * time.Second
	authTimeout = 5 * time.Second
)

// errNoResponse means the server never started answering. Distinct from an
// empty answer, and not worth reconnecting over.
var errNoResponse = errors.New("no response from the server")

// errDesync means a packet stopped part way through, so the stream is no
// longer aligned to packet boundaries and the connection cannot be reused.
var errDesync = errors.New("response cut short")

type Rcon struct {
	mu       sync.Mutex
	conn     net.Conn
	addr     string
	password string
	nextID   int32
}

func New(host string, port int, password string) *Rcon {
	return &Rcon{addr: fmt.Sprintf("%s:%d", host, port), password: password}
}

// Connect dials the server and authenticates. Safe to call again to reconnect.
func (r *Rcon) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connectLocked()
}

func (r *Rcon) connectLocked() error {
	r.closeLocked()
	conn, err := net.DialTimeout("tcp", r.addr, dialTimeout)
	if err != nil {
		return err
	}
	r.nextID = 1
	if err := writePacket(conn, r.nextID, packetAuth, r.password); err != nil {
		conn.Close()
		return err
	}
	// The server may send an empty RESPONSE_VALUE before the auth response.
	for {
		id, typ, _, err := readPacket(conn, authTimeout, authTimeout)
		if err != nil {
			conn.Close()
			if errors.Is(err, io.EOF) {
				// CS2 drops the connection instead of answering id -1.
				return errors.New("server rejected the RCON password — did it change after a server restart? Update it in the settings")
			}
			return err
		}
		if typ != packetAuthResponse {
			continue
		}
		if id == -1 {
			conn.Close()
			return errors.New("wrong RCON password")
		}
		r.conn = conn
		return nil
	}
}

// Exec sends a command and returns the server's full response.
func (r *Rcon) Exec(command string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out, err := r.execLocked(command)
	if err == nil || errors.Is(err, errNoResponse) {
		// A silent server is reported as such: reconnecting would only make
		// the caller wait for the same silence twice.
		return out, err
	}
	// A broken or out-of-step connection: rebuild it and try once more.
	if err := r.connectLocked(); err != nil {
		return "", err
	}
	return r.execLocked(command)
}

func (r *Rcon) execLocked(command string) (string, error) {
	if r.conn == nil {
		return "", errors.New("not connected")
	}

	r.nextID++
	id := r.nextID
	if err := writePacket(r.conn, id, packetExec, command); err != nil {
		r.closeLocked()
		return "", err
	}

	var out bytes.Buffer
	started := false
	for {
		wait := responseStartWait
		if started {
			wait = responseQuietWait
		}
		packetID, _, body, err := readPacket(r.conn, wait, packetBodyWait)
		if err != nil {
			if errors.Is(err, errDesync) {
				r.closeLocked()
				return "", err
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				if started {
					return out.String(), nil // quiet: the answer is complete
				}
				return "", fmt.Errorf("%w: %q got nothing back within %s",
					errNoResponse, command, responseStartWait)
			}
			r.closeLocked()
			return "", err
		}
		// Ids count up, so anything below this command's is output from one
		// we already finished with — CS2 sends it whenever it gets round to
		// it. Ids we never issued (0, -1) are left alone.
		if packetID > 0 && packetID < id {
			continue
		}
		started = true
		out.WriteString(body)
	}
}

func (r *Rcon) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
}

func (r *Rcon) closeLocked() {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}

func writePacket(conn net.Conn, id, typ int32, body string) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, int32(len(body)+10))
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, typ)
	buf.WriteString(body)
	buf.Write([]byte{0, 0})
	_, err := conn.Write(buf.Bytes())
	return err
}

// readPacket reads one packet. headerWait bounds how long to wait for a
// packet to start arriving; once its header is in, the body is already on the
// way and gets bodyWait instead. Keeping the two apart is what stops a slow
// link from cutting a long answer in half.
func readPacket(conn net.Conn, headerWait, bodyWait time.Duration) (id, typ int32, body string, err error) {
	if err = conn.SetReadDeadline(time.Now().Add(headerWait)); err != nil {
		return
	}
	var size int32
	if err = binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return
	}
	if size < 10 || size > maxPacketSize {
		err = fmt.Errorf("%w: invalid packet size %d", errDesync, size)
		return
	}

	if err = conn.SetReadDeadline(time.Now().Add(bodyWait)); err != nil {
		return
	}
	data := make([]byte, size)
	if _, err = io.ReadFull(conn, data); err != nil {
		err = fmt.Errorf("%w: %v", errDesync, err)
		return
	}

	id = int32(binary.LittleEndian.Uint32(data[0:4]))
	typ = int32(binary.LittleEndian.Uint32(data[4:8]))
	body = string(bytes.TrimRight(data[8:], "\x00"))
	return
}
