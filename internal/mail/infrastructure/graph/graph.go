// Package graph is the domain.Backend for Outlook.com and Microsoft 365: the mail
// of an account through Microsoft Graph, over plain HTTP and JSON (the official SDK
// is huge and needs a newer Go than the reader's).
//
// The access token comes from a function, so the Accounts context renews it when it
// expires; a 401 is reported as domain.ErrUnauthorized for the caller to renew and
// try again.
package graph

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/mail/infrastructure/mime"
)

// DefaultBase is Microsoft Graph's address.
const DefaultBase = "https://graph.microsoft.com/v1.0"

// Limits: the reader has 1 GB for everything.
const (
	maxJSON    = 4 << 20  // a folder list or a page of messages
	maxMessage = 10 << 20 // one message with its attachments: a bigger one is not opened here
	maxPage    = 100      // messages per request
)

// Doer is what the client needs of an HTTP client (*http.Client satisfies it).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client implements domain.Backend over Microsoft Graph.
type Client struct {
	HTTP  Doer
	Token func() (string, error) // the current access token
	Base  string                 // DefaultBase when empty
}

var _ domain.Backend = Client{}

// wellKnown maps Graph's folder aliases to roles.
var wellKnown = []struct {
	alias string
	role  domain.FolderRole
}{
	{"inbox", domain.Inbox}, {"sentitems", domain.Sent}, {"drafts", domain.Drafts},
	{"deleteditems", domain.Trash}, {"junkemail", domain.Junk}, {"archive", domain.Archive},
}

// messagePath is the address of one message.
func messagePath(remoteID string) string { return "/me/messages/" + url.PathEscape(remoteID) }

func (c Client) base() string {
	if c.Base == "" {
		return DefaultBase
	}
	return c.Base
}

// do sends a request and returns the body of a successful answer.
func (c Client) do(method, path, contentType string, body []byte, limit int64) ([]byte, error) {
	token, err := c.Token()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.base()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("graph: the answer is larger than %d bytes: too big to open on the reader", limit)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, domain.ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, graphError(resp.StatusCode, data)
	}
	return data, nil
}

// graphError is the error of a failed answer: Graph's code and message, never a token.
func graphError(status int, body []byte) error {
	var e struct {
		Error struct{ Code, Message string }
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Code != "" {
		return fmt.Errorf("graph: %s: %s", e.Error.Code, firstLine(e.Error.Message))
	}
	return fmt.Errorf("graph: HTTP %d", status)
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

type graphFolder struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// Folders implements domain.Backend: the well-known folders (to know the roles) and the rest.
func (c Client) Folders() ([]domain.Folder, error) {
	roles := map[string]domain.FolderRole{}
	for _, w := range wellKnown {
		data, err := c.do(http.MethodGet, "/me/mailFolders/"+w.alias+"?$select=id,displayName", "", nil, maxJSON)
		if err == domain.ErrUnauthorized {
			return nil, err
		}
		if err != nil {
			continue // an account may have no Archive folder
		}
		var f graphFolder
		if json.Unmarshal(data, &f) == nil && f.ID != "" {
			roles[f.ID] = w.role
		}
	}
	var out []domain.Folder
	path := "/me/mailFolders?$top=100&$select=id,displayName"
	for page := 0; path != "" && page < 5; page++ { // five pages of 100 folders is more than anyone has
		data, err := c.do(http.MethodGet, path, "", nil, maxJSON)
		if err != nil {
			return nil, err
		}
		var list struct {
			Value    []graphFolder `json:"value"`
			NextLink string        `json:"@odata.nextLink"`
		}
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("graph: unexpected answer: %w", err)
		}
		for _, f := range list.Value {
			role, ok := roles[f.ID]
			if !ok {
				role = domain.Other
			}
			out = append(out, domain.Folder{ID: f.ID, Name: f.DisplayName, Role: role})
		}
		path = strings.TrimPrefix(list.NextLink, c.base())
		if !strings.HasPrefix(path, "/") {
			path = ""
		}
	}
	return out, nil
}

type emailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

func addressOf(a emailAddress) domain.Address {
	return domain.Address{Name: a.EmailAddress.Name, Email: a.EmailAddress.Address}
}

func addressesOf(list []emailAddress) []domain.Address {
	out := make([]domain.Address, 0, len(list))
	for _, a := range list {
		out = append(out, addressOf(a))
	}
	return out
}

const messageFields = "id,internetMessageId,subject,bodyPreview,receivedDateTime,isRead,hasAttachments,flag,from,toRecipients,ccRecipients,replyTo"

// Recent implements domain.Backend.
func (c Client) Recent(folderID string, limit int) ([]domain.Message, error) {
	if limit <= 0 || limit > maxPage {
		limit = maxPage
	}
	q := url.Values{}
	q.Set("$top", fmt.Sprint(limit))
	q.Set("$orderby", "receivedDateTime desc")
	q.Set("$select", messageFields)
	data, err := c.do(http.MethodGet, "/me/mailFolders/"+url.PathEscape(folderID)+"/messages?"+q.Encode(), "", nil, maxJSON)
	if err != nil {
		return nil, err
	}
	var list struct {
		Value []struct {
			ID             string    `json:"id"`
			InternetID     string    `json:"internetMessageId"`
			Subject        string    `json:"subject"`
			Preview        string    `json:"bodyPreview"`
			Received       time.Time `json:"receivedDateTime"`
			IsRead         bool      `json:"isRead"`
			HasAttachments bool      `json:"hasAttachments"`
			Flag           struct {
				FlagStatus string `json:"flagStatus"`
			} `json:"flag"`
			From         emailAddress   `json:"from"`
			ToRecipients []emailAddress `json:"toRecipients"`
			CcRecipients []emailAddress `json:"ccRecipients"`
			ReplyTo      []emailAddress `json:"replyTo"`
		} `json:"value"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("graph: unexpected answer: %w", err)
	}
	out := make([]domain.Message, 0, len(list.Value))
	for _, m := range list.Value {
		msg := domain.Message{
			FolderID: folderID, RemoteID: m.ID, MessageID: m.InternetID, Subject: m.Subject, Preview: strings.TrimSpace(m.Preview),
			Date: m.Received.UTC(), From: addressOf(m.From), To: addressesOf(m.ToRecipients), Cc: addressesOf(m.CcRecipients),
			ReplyTo: addressesOf(m.ReplyTo),
		}
		if m.IsRead {
			msg.Flags |= domain.Seen
		}
		if strings.EqualFold(m.Flag.FlagStatus, "flagged") {
			msg.Flags |= domain.Flagged
		}
		if m.HasAttachments { // the names come with the full message
			msg.Attachments = []domain.Attachment{{Name: "…"}}
		}
		out = append(out, msg)
	}
	return out, nil
}

// Fetch implements domain.Backend: the whole message as MIME, read like any other.
func (c Client) Fetch(folderID, remoteID string) (domain.Message, error) {
	data, err := c.do(http.MethodGet, messagePath(remoteID)+"/$value", "", nil, maxMessage)
	if err != nil {
		return domain.Message{}, err
	}
	m, err := mime.Parse(bytes.NewReader(data))
	if err != nil {
		return domain.Message{}, err
	}
	m.FolderID, m.RemoteID = folderID, remoteID
	return m, nil
}

// Send implements domain.Backend: Graph accepts the message as MIME, base64-encoded.
func (c Client) Send(d domain.Draft) error {
	raw, err := mime.Compose(d, time.Now(), mime.NewMessageID(d.From))
	if err != nil {
		return err
	}
	// Graph reads the recipients from the MIME headers. mime.Compose leaves Bcc out on purpose,
	// so a draft with Bcc recipients gets the header added here; Graph is expected to deliver
	// to them without showing the header to the others (not yet checked against the real service).
	if len(d.Bcc) > 0 {
		raw = withBcc(raw, d.Bcc)
	}
	body := []byte(base64.StdEncoding.EncodeToString(raw))
	_, err = c.do(http.MethodPost, "/me/sendMail", "text/plain", body, maxJSON)
	return err
}

// withBcc adds a Bcc header to the message for the server to read.
func withBcc(raw []byte, bcc []domain.Address) []byte {
	parts := make([]string, len(bcc))
	for i, a := range bcc {
		parts[i] = a.String()
	}
	return append([]byte("Bcc: "+strings.Join(parts, ", ")+"\r\n"), raw...)
}

// SetRead implements domain.Backend.
func (c Client) SetRead(_, remoteID string, read bool) error {
	body, _ := json.Marshal(map[string]bool{"isRead": read})
	_, err := c.do(http.MethodPatch, messagePath(remoteID), "application/json", body, maxJSON)
	return err
}

// SetFlagged implements domain.Backend.
func (c Client) SetFlagged(_, remoteID string, flagged bool) error {
	status := "notFlagged"
	if flagged {
		status = "flagged"
	}
	body, _ := json.Marshal(map[string]map[string]string{"flag": {"flagStatus": status}})
	_, err := c.do(http.MethodPatch, messagePath(remoteID), "application/json", body, maxJSON)
	return err
}

// Delete implements domain.Backend: Graph moves the message to Deleted Items.
func (c Client) Delete(_, remoteID string) error {
	_, err := c.do(http.MethodDelete, messagePath(remoteID), "", nil, maxJSON)
	return err
}
