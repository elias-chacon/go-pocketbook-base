package smtpsend

import (
	"bufio"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"testing"
)

// fakeServer speaks just enough SMTP to receive one message.
type fakeServer struct {
	ln      net.Listener
	mu      sync.Mutex
	from    string
	rcpts   []string
	data    string
	authed  string
	rejectR string
}

func newServer(t *testing.T) *fakeServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{ln: ln}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(c)
		}
	}()
	return s
}

// readData reads the message up to the line with a single dot.
func readData(r *bufio.Reader) string {
	var b strings.Builder
	for {
		l, err := r.ReadString('\n')
		if err != nil || l == ".\r\n" {
			return b.String()
		}
		b.WriteString(l)
	}
}

func (s *fakeServer) serve(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	say := func(l string) { c.Write([]byte(l + "\r\n")) }
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		up := strings.ToUpper(line)
		s.mu.Lock()
		switch {
		case strings.HasPrefix(up, "EHLO"):
			say("250-fake\r\n250 AUTH PLAIN XOAUTH2")
		case strings.HasPrefix(up, "AUTH"):
			s.authed = line
			say("235 ok")
		case strings.HasPrefix(up, "MAIL FROM:"):
			s.from = line
			say("250 ok")
		case strings.HasPrefix(up, "RCPT TO:"):
			if s.rejectR != "" && strings.Contains(line, s.rejectR) {
				say("550 no such user")
			} else {
				s.rcpts = append(s.rcpts, line)
				say("250 ok")
			}
		case up == "DATA":
			say("354 go")
			s.data = readData(r)
			say("250 queued")
		case up == "QUIT":
			say("221 bye")
			s.mu.Unlock()
			return
		default:
			say("250 ok")
		}
		s.mu.Unlock()
	}
}

func TestSendDeliversTheEnvelopeAndTheMessage(t *testing.T) {
	srv := newServer(t)
	snd := Sender{Addr: srv.ln.Addr().String(), Security: NoTLS}
	err := snd.Send("ana@example.org", []string{"bia@example.org", "cris@example.org"}, []byte("Subject: oi\r\n\r\ncorpo\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(srv.from, "ana@example.org") || len(srv.rcpts) != 2 {
		t.Errorf("envelope: %q %q", srv.from, srv.rcpts)
	}
	if !strings.Contains(srv.data, "Subject: oi") || !strings.Contains(srv.data, "corpo") {
		t.Errorf("data = %q", srv.data)
	}
}

func TestSendSignsInWhenAsked(t *testing.T) {
	srv := newServer(t)
	snd := Sender{Addr: srv.ln.Addr().String(), Security: NoTLS, Auth: smtp.PlainAuth("", "ana", "pw", "127.0.0.1")}
	if err := snd.Send("ana@example.org", []string{"bia@example.org"}, []byte("x\r\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(srv.authed, "AUTH PLAIN") {
		t.Errorf("no sign-in: %q", srv.authed)
	}
}

func TestSendReportsARefusedRecipient(t *testing.T) {
	srv := newServer(t)
	srv.rejectR = "ruim@"
	snd := Sender{Addr: srv.ln.Addr().String(), Security: NoTLS}
	err := snd.Send("ana@example.org", []string{"ruim@example.org"}, []byte("x\r\n"))
	if err == nil || !strings.Contains(err.Error(), "ruim@example.org") {
		t.Errorf("err = %v", err)
	}
}

func TestSendNeedsASenderAndARecipient(t *testing.T) {
	if (Sender{Addr: "127.0.0.1:1"}).Send("", []string{"a@b"}, nil) == nil {
		t.Error("no sender accepted")
	}
	if (Sender{Addr: "127.0.0.1:1"}).Send("a@b", nil, nil) == nil {
		t.Error("no recipient accepted")
	}
}

func TestSendFailsClearlyWhenTheServerIsDown(t *testing.T) {
	if err := (Sender{Addr: "127.0.0.1:1", Security: NoTLS}).Send("a@b", []string{"c@d"}, nil); err == nil || !strings.Contains(err.Error(), "cannot reach") {
		t.Errorf("err = %v", err)
	}
	if err := (Sender{Addr: "sem-porta"}).Send("a@b", []string{"c@d"}, nil); err == nil {
		t.Error("bad address accepted")
	}
}
