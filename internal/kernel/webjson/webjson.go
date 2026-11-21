// Package webjson is the small HTTP+JSON client the to-do service adapters share: a bearer
// token from a function (so the Accounts context renews it), a JSON body in and out, and an
// error that carries the HTTP status. It depends on nothing.
package webjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxAnswer bounds what is read from one answer: the reader has 1 GB for everything.
const maxAnswer = 4 << 20

// Doer is what the client needs of an HTTP client (*http.Client satisfies it).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// StatusError is an answer that was not a success.
type StatusError struct {
	Code    int
	Message string
}

func (e *StatusError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("HTTP %d", e.Code)
}

// Client sends JSON requests to one service.
type Client struct {
	HTTP  Doer
	Token func() (string, error) // nil for a public service: no Authorization header is sent
	Base  string                 // put before a path; a path that is already an address is used as it is
	// Header is sent with every request (Graph's Prefer, for instance).
	Header map[string]string
}

// Do sends a request. in (a value, or nil) is the JSON body; out (a pointer, or nil)
// receives the JSON answer.
func (c Client) Do(method, path string, in, out interface{}) error {
	token := ""
	if c.Token != nil { // a public service needs none
		var err error
		if token, err = c.Token(); err != nil {
			return err
		}
	}
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	target := path
	if !strings.HasPrefix(path, "https://") && !strings.HasPrefix(path, "http://") {
		target = c.Base + path
	}
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		return err
	}
	if c.Token != nil {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range c.Header {
		req.Header.Set(k, v)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &StatusError{Code: resp.StatusCode, Message: message(data)}
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// message finds the service's own words in an error answer ({"error":{"message":…}}).
func message(data []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &e) == nil {
		return e.Error.Message
	}
	return ""
}
