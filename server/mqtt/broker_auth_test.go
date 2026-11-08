package mqtt

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"boivie/ubergang/server/log"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

func TestLoadBrokerCredentials(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	withNewline := write("pw", "s3cret\n")
	empty := write("empty", "\n")

	if c, err := LoadBrokerCredentials("", ""); err != nil || c != (BrokerCredentials{}) {
		t.Errorf("no flags: got %+v, %v; want anonymous", c, err)
	}
	if c, err := LoadBrokerCredentials("ubergang", withNewline); err != nil || c != (BrokerCredentials{"ubergang", "s3cret"}) {
		t.Errorf("both flags: got %+v, %v", c, err)
	}
	for _, tc := range []struct{ name, user, file string }{
		{"username only", "ubergang", ""},
		{"file only", "", withNewline},
		{"missing file", "ubergang", filepath.Join(dir, "nope")},
		{"empty file", "ubergang", empty},
	} {
		if _, err := LoadBrokerCredentials(tc.user, tc.file); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
}

// roundTrip encodes a CONNECT the way it's sent to the broker and decodes it
// again, as the broker would.
func roundTrip(t *testing.T, c *packets.ConnectPacket) *packets.ConnectPacket {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Write(&buf); err != nil {
		t.Fatal(err)
	}
	p, err := packets.ReadPacket(&buf)
	if err != nil {
		t.Fatal(err)
	}
	return p.(*packets.ConnectPacket)
}

func clientConnect() *packets.ConnectPacket {
	c := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
	c.ProtocolName, c.ProtocolVersion = "MQTT", 4
	c.ClientIdentifier = "shellyplug-s-26748E"
	c.UsernameFlag, c.Username = true, "shelly-sanglampa"
	c.PasswordFlag, c.Password = true, []byte("client-password")
	c.WillFlag, c.WillTopic, c.WillMessage = true, "shellies/x/online", []byte("false")
	return c
}

func TestRewriteConnectWithCredentials(t *testing.T) {
	c := clientConnect()
	rewriteConnectForBroker(c, "shelly-sanglampa", BrokerCredentials{"ubergang", "upstream-pw"})
	got := roundTrip(t, c)

	if got.ClientIdentifier != "ug-shelly-sanglampa" {
		t.Errorf("client id = %q", got.ClientIdentifier)
	}
	if !got.UsernameFlag || got.Username != "ubergang" || !got.PasswordFlag || string(got.Password) != "upstream-pw" {
		t.Errorf("credentials = %v %q / %v %q, want ubergang's", got.UsernameFlag, got.Username, got.PasswordFlag, got.Password)
	}
	if !got.WillFlag || got.WillTopic != "shellies/x/online" || string(got.WillMessage) != "false" {
		t.Errorf("will not preserved: %v %q %q", got.WillFlag, got.WillTopic, got.WillMessage)
	}
}

func TestRewriteConnectAnonymous(t *testing.T) {
	// The client's own credentials must never reach the broker, flags included.
	c := clientConnect()
	rewriteConnectForBroker(c, "shelly-sanglampa", BrokerCredentials{})
	got := roundTrip(t, c)

	if got.UsernameFlag || got.PasswordFlag || got.Username != "" || len(got.Password) != 0 {
		t.Errorf("anonymous CONNECT carries credentials: %v %q / %v %q", got.UsernameFlag, got.Username, got.PasswordFlag, got.Password)
	}
	if got.ClientIdentifier != "ug-shelly-sanglampa" {
		t.Errorf("client id = %q", got.ClientIdentifier)
	}
}

// TestPublisherSendsCredentials runs the publisher against a fake broker and
// checks the CONNECT it receives.
func TestPublisherSendsCredentials(t *testing.T) {
	for _, tc := range []struct {
		name  string
		creds BrokerCredentials
	}{
		{"authenticated", BrokerCredentials{"ubergang", "upstream-pw"}},
		{"anonymous", BrokerCredentials{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ln.Close() }()
			got := make(chan *packets.ConnectPacket, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				defer func() { _ = conn.Close() }()
				p, err := packets.ReadPacket(conn)
				if err != nil {
					return
				}
				got <- p.(*packets.ConnectPacket)
				ack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
				_ = ack.Write(conn)
				time.Sleep(200 * time.Millisecond)
			}()

			pub := NewPublisher(log.NewLogger(log.Fields{}), ln.Addr().String(), tc.creds)
			defer func() { _ = pub.Close() }()

			select {
			case c := <-got:
				if c.UsernameFlag != (tc.creds.Username != "") || c.Username != tc.creds.Username ||
					c.PasswordFlag != (tc.creds.Password != "") || string(c.Password) != tc.creds.Password {
					t.Errorf("broker got %v %q / %v %q, want %+v", c.UsernameFlag, c.Username, c.PasswordFlag, c.Password, tc.creds)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("publisher never connected")
			}
		})
	}
}
