package xoauth2

import (
	"errors"
	"net/smtp"
	"strings"
	"testing"
)

func auth(token string, err error) Auth {
	return Auth{User: "ana@example.org", Token: func() (string, error) { return token, err }}
}

func TestInitialResponse(t *testing.T) {
	mech, ir, err := auth("tok", nil).SASL().Start()
	if err != nil || mech != "XOAUTH2" {
		t.Fatalf("Start = %q, %v", mech, err)
	}
	if string(ir) != "user=ana@example.org\x01auth=Bearer tok\x01\x01" {
		t.Errorf("initial response = %q", ir)
	}
	proto, ir2, err := auth("tok", nil).SMTP().Start(&smtp.ServerInfo{})
	if err != nil || proto != "XOAUTH2" || string(ir2) != string(ir) {
		t.Errorf("SMTP Start = %q %q %v", proto, ir2, err)
	}
}

func TestNoTokenFails(t *testing.T) {
	if _, _, err := auth("", nil).SASL().Start(); !errors.Is(err, ErrNoToken) {
		t.Errorf("empty token: %v", err)
	}
	boom := errors.New("renew failed")
	if _, _, err := auth("", boom).SMTP().Start(nil); !errors.Is(err, boom) {
		t.Errorf("token error not passed on: %v", err)
	}
}

func TestServerErrorIsAnsweredEmpty(t *testing.T) {
	out, err := auth("tok", nil).SASL().Next([]byte(`{"status":"401"}`))
	if err != nil || out == nil || len(out) != 0 {
		t.Errorf("sasl Next = %v, %v", out, err)
	}
	out, err = auth("tok", nil).SMTP().Next([]byte("x"), true)
	if err != nil || out == nil || len(out) != 0 {
		t.Errorf("smtp Next(more) = %v, %v", out, err)
	}
	if out, _ := auth("tok", nil).SMTP().Next(nil, false); out != nil {
		t.Errorf("smtp Next(done) = %v", out)
	}
}

func TestTokenNeverInErrors(t *testing.T) {
	_, _, err := auth("SECRET", errors.New("boom")).SASL().Start()
	if err != nil && strings.Contains(err.Error(), "SECRET") {
		t.Error("token leaked")
	}
}
