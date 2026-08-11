package rcon

// How the client copes with a server that is slow, chatty, or both. Every
// case here was a silent wrong answer before: an empty map list with no error
// to explain it.

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

// writePacketSlowly sends a packet's header, then its body after a pause, the
// way a slow link delivers one.
func writePacketSlowly(conn net.Conn, id, typ int32, body string, pause time.Duration) error {
	head := new(bytes.Buffer)
	binary.Write(head, binary.LittleEndian, int32(len(body)+10))
	binary.Write(head, binary.LittleEndian, id)
	binary.Write(head, binary.LittleEndian, typ)
	if _, err := conn.Write(head.Bytes()); err != nil {
		return err
	}
	time.Sleep(pause)
	_, err := conn.Write(append([]byte(body), 0, 0))
	return err
}

// fakeServer stands in for CS2's RCON. It accepts any password, then hands
// each command to handle, which may answer late, in pieces, or with leftovers
// from an earlier command.
func fakeServer(t *testing.T, handle func(conn net.Conn, id int32, cmd string)) *Rcon {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			id, typ, body, err := readPacket(conn, 30*time.Second, 30*time.Second)
			if err != nil {
				return
			}
			if typ == packetAuth {
				writePacket(conn, id, packetAuthResponse, "")
				continue
			}
			handle(conn, id, body)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	r := New("127.0.0.1", addr.Port, "pw")
	if err := r.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(r.Close)
	return r
}

// A heavy command such as `maps *` can take a server seconds to answer. The
// answer must still arrive, not be cut off by an impatient client.
func TestExecWaitsForASlowServer(t *testing.T) {
	r := fakeServer(t, func(conn net.Conn, id int32, cmd string) {
		time.Sleep(3 * time.Second)
		writePacket(conn, id, packetResponse, "de_dust2\nde_inferno\n")
	})

	out, err := r.Exec("maps *")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.Contains(out, "de_inferno") {
		t.Errorf("got %q, want the full answer", out)
	}
}

// A server that never answers has to say so. Returning "" with a nil error
// was indistinguishable from an empty response, which is how the map list
// came back empty with nothing reported anywhere.
func TestExecReportsSilence(t *testing.T) {
	r := fakeServer(t, func(conn net.Conn, id int32, cmd string) {})

	out, err := r.Exec("maps *")
	if err == nil {
		t.Fatalf("expected an error, got %q with no error", out)
	}
	if out != "" {
		t.Errorf("got %q, want no output alongside the error", out)
	}
}

// CS2 answers asynchronously, so an earlier command's output can still be in
// flight when the next one is sent. Responses carry the request id; without
// checking it, one command reads another's answer.
func TestExecIgnoresAnEarlierCommandsOutput(t *testing.T) {
	r := fakeServer(t, func(conn net.Conn, id int32, cmd string) {
		if id > 2 {
			// Leftovers from the previous command, arriving late.
			writePacket(conn, id-1, packetResponse, "hostname : leftover status output\n")
		}
		writePacket(conn, id, packetResponse, "de_dust2\n")
	})

	if _, err := r.Exec("status"); err != nil {
		t.Fatalf("first Exec: %v", err)
	}
	out, err := r.Exec("maps *")
	if err != nil {
		t.Fatalf("second Exec: %v", err)
	}
	if strings.Contains(out, "leftover") {
		t.Errorf("got %q, which contains the previous command's output", out)
	}
	if !strings.Contains(out, "de_dust2") {
		t.Errorf("got %q, want this command's own answer", out)
	}
}

// A packet already has its header in the socket, so its body is on its way
// even if the link is slow. Bounding the body by the short between-packets
// wait truncated long answers without a word.
func TestExecDoesNotTruncateASlowPacketBody(t *testing.T) {
	tail := strings.Repeat("de_somemap\n", 200)
	r := fakeServer(t, func(conn net.Conn, id int32, cmd string) {
		writePacket(conn, id, packetResponse, "first\n")
		time.Sleep(50 * time.Millisecond)
		// Second packet, its body dribbling in well past the quiet period.
		writePacketSlowly(conn, id, packetResponse, tail, 600*time.Millisecond)
	})

	out, err := r.Exec("maps *")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if !strings.HasPrefix(out, "first\n") {
		t.Fatalf("got %q, want it to start with the first packet", out)
	}
	if len(out) < len(tail) {
		t.Errorf("got %d bytes, want the whole %d-byte answer", len(out), len(tail)+6)
	}
}
