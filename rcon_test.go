package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestPacketRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		writePacket(client, 7, packetExec, "status")
	}()

	server.SetReadDeadline(time.Now().Add(2 * time.Second))
	id, typ, body, err := readPacket(server)
	if err != nil {
		t.Fatalf("readPacket: %v", err)
	}
	if id != 7 || typ != packetExec || body != "status" {
		t.Errorf("got id=%d typ=%d body=%q, want 7/%d/status", id, typ, body, packetExec)
	}
}

func TestReadPacketRejectsAbsurdSize(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		// A size field far beyond anything a server sends: reading it as a
		// length would otherwise try to allocate it.
		binary.Write(client, binary.LittleEndian, int32(1<<30))
	}()

	server.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, _, err := readPacket(server); err == nil {
		t.Fatal("expected an error for an out-of-range packet size")
	}
}

// CS2 closes the connection instead of answering with id -1 when the password
// is wrong. That used to surface as a bare "EOF", which told the user nothing.
func TestConnectWrongPasswordExplainsItself(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		readPacket(conn) // the auth attempt
		conn.Close()     // ...and hang up, as CS2 does
	}()

	addr := listener.Addr().(*net.TCPAddr)
	r := NewRcon("127.0.0.1", addr.Port, "wrong")
	err = r.Connect()
	if err == nil {
		t.Fatal("expected an error when the server hangs up on auth")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("error %q should mention the password", err)
	}
}

func TestExecWithoutConnection(t *testing.T) {
	r := NewRcon("127.0.0.1", 1, "pw")
	if _, err := r.Exec("status"); err == nil {
		t.Fatal("expected an error when there is no connection")
	}
}

// writePacket must produce the layout CS2 expects: size, id, type, body, two
// trailing nulls.
func TestWritePacketLayout(t *testing.T) {
	var buf bytes.Buffer
	conn := &bufferConn{Writer: &buf}
	if err := writePacket(conn, 3, packetAuth, "secret"); err != nil {
		t.Fatalf("writePacket: %v", err)
	}
	raw := buf.Bytes()
	size := int32(binary.LittleEndian.Uint32(raw[0:4]))
	if int(size) != len("secret")+10 {
		t.Errorf("size = %d, want %d", size, len("secret")+10)
	}
	if got := binary.LittleEndian.Uint32(raw[4:8]); got != 3 {
		t.Errorf("id = %d, want 3", got)
	}
	if got := binary.LittleEndian.Uint32(raw[8:12]); got != packetAuth {
		t.Errorf("type = %d, want %d", got, packetAuth)
	}
	if !bytes.HasSuffix(raw, []byte{0, 0}) {
		t.Error("packet must end with two null bytes")
	}
}

// bufferConn is a net.Conn that only supports writing to a buffer.
type bufferConn struct {
	net.Conn
	Writer *bytes.Buffer
}

func (c *bufferConn) Write(p []byte) (int, error) { return c.Writer.Write(p) }
func (c *bufferConn) Close() error                { return nil }
func (c *bufferConn) SetReadDeadline(time.Time) error {
	return errors.New("not supported")
}
