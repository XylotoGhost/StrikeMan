package main

// Minimal client for the Source RCON protocol (used by CS2).
// Wire format per packet: int32 size, int32 id, int32 type, body, two null bytes.

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
)

type Rcon struct {
	mu       sync.Mutex
	conn     net.Conn
	addr     string
	password string
	nextID   int32
}

func NewRcon(host string, port int, password string) *Rcon {
	return &Rcon{addr: fmt.Sprintf("%s:%d", host, port), password: password}
}

// Connect dials the server and authenticates. Safe to call again to reconnect.
func (r *Rcon) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connectLocked()
}

func (r *Rcon) connectLocked() error {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	conn, err := net.DialTimeout("tcp", r.addr, 5*time.Second)
	if err != nil {
		return err
	}
	r.nextID = 1
	if err := writePacket(conn, r.nextID, packetAuth, r.password); err != nil {
		conn.Close()
		return err
	}
	// The server may send an empty RESPONSE_VALUE before the auth response.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		id, typ, _, err := readPacket(conn)
		if err != nil {
			conn.Close()
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
// It reconnects and retries once if the connection has gone stale.
func (r *Rcon) Exec(command string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.execLocked(command)
	if err != nil {
		if err := r.connectLocked(); err != nil {
			return "", err
		}
		return r.execLocked(command)
	}
	return out, nil
}

func (r *Rcon) execLocked(command string) (string, error) {
	if r.conn == nil {
		return "", errors.New("not connected")
	}
	// Drain packets left over from a previous command — CS2 sends command
	// output asynchronously, sometimes after we already stopped reading.
	r.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	for {
		if _, _, _, err := readPacket(r.conn); err != nil {
			break
		}
	}

	r.nextID++
	if err := writePacket(r.conn, r.nextID, packetExec, command); err != nil {
		return "", err
	}

	// CS2 responds with one or more RESPONSE_VALUE packets, but there is no
	// reliable end marker: read until a short period of silence.
	var out bytes.Buffer
	deadline := 2 * time.Second // generous wait for the first packet
	for {
		r.conn.SetReadDeadline(time.Now().Add(deadline))
		_, _, body, err := readPacket(r.conn)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return out.String(), nil
			}
			return "", err
		}
		out.WriteString(body)
		deadline = 300 * time.Millisecond
	}
}

func (r *Rcon) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
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

func readPacket(conn net.Conn) (id, typ int32, body string, err error) {
	var size int32
	if err = binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return
	}
	if size < 10 || size > 1<<20 {
		err = fmt.Errorf("invalid packet size %d", size)
		return
	}
	data := make([]byte, size)
	if _, err = io.ReadFull(conn, data); err != nil {
		return
	}
	id = int32(binary.LittleEndian.Uint32(data[0:4]))
	typ = int32(binary.LittleEndian.Uint32(data[4:8]))
	body = string(bytes.TrimRight(data[8:], "\x00"))
	return
}
