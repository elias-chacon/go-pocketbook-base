// Package mime is the reader and the writer of e-mail on the wire (RFC 5322 and
// MIME): it turns the bytes of a message into the domain's Message, and a Draft
// into the bytes to send. Charsets, encoded words, quoted-printable and base64 are
// handled here, so the rest of the code only sees text.
package mime

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf8"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"

	"go-pocketbook-base/internal/mail/domain"
)

func init() {
	// A message in a charset we do not know is shown as it is, not refused.
	message.CharsetReader = func(name string, input io.Reader) (io.Reader, error) {
		if r, err := charset.Reader(name, input); err == nil {
			return r, nil
		}
		return input, nil
	}
}

// Limits of what is read into memory: the reader has 1 GB for everything.
const (
	maxTextPart = 2 << 20 // the text of a message
	previewLen  = 140     // characters of the list preview
)

// Parse reads the bytes of a message. The Message has no ID, AccountID, FolderID or
// RemoteID: those belong to where it was fetched from.
func Parse(r io.Reader) (domain.Message, error) {
	mr, err := mail.CreateReader(r)
	if err != nil && !message.IsUnknownCharset(err) {
		return domain.Message{}, err
	}
	if mr == nil {
		return domain.Message{}, errors.New("mime: not a message")
	}
	defer mr.Close()
	if mr.Header.Len() == 0 {
		return domain.Message{}, errors.New("mime: not a message: it has no header")
	}

	m := headerOf(mr.Header)
	plain, htmlBody, err := readParts(mr, &m)
	if err != nil {
		return domain.Message{}, err
	}
	switch {
	case strings.TrimSpace(plain) != "":
		m.Body = strings.TrimSpace(strings.ReplaceAll(plain, "\r\n", "\n"))
	case htmlBody != "":
		m.Body = htmlToText(htmlBody)
	}
	m.Preview = preview(m.Body)
	return m, nil
}

// headerOf reads the fields of the header. A field that cannot be read is left
// empty: one bad header must not hide the message.
func headerOf(h mail.Header) domain.Message {
	var m domain.Message
	if from, _ := h.AddressList("From"); len(from) > 0 {
		m.From = addressOf(from[0])
	}
	m.ReplyTo = addressesOf(h, "Reply-To")
	m.To, m.Cc = addressesOf(h, "To"), addressesOf(h, "Cc")
	m.Subject, _ = h.Subject()
	m.Date, _ = h.Date()
	if id, _ := h.MessageID(); id != "" {
		m.MessageID = "<" + id + ">"
	}
	if ids, _ := h.MsgIDList("In-Reply-To"); len(ids) > 0 {
		m.InReplyTo = "<" + ids[0] + ">"
	}
	refs, _ := h.MsgIDList("References")
	for _, id := range refs {
		m.References = append(m.References, "<"+id+">")
	}
	return m
}

func addressesOf(h mail.Header, key string) []domain.Address {
	list, _ := h.AddressList(key)
	out := make([]domain.Address, 0, len(list))
	for _, a := range list {
		out = append(out, addressOf(a))
	}
	return out
}

func addressOf(a *mail.Address) domain.Address {
	return domain.Address{Name: a.Name, Email: a.Address}
}

// bodies are the text parts found so far: the first of each kind wins.
type bodies struct{ plain, html string }

func (b *bodies) add(kind, text string) {
	switch kind {
	case "text/plain", "":
		if b.plain == "" {
			b.plain = text
		}
	case "text/html":
		if b.html == "" {
			b.html = text
		}
	}
}

// readParts walks the parts: the text and the HTML bodies, and the attachments
// (their names and sizes, not their content).
func readParts(mr *mail.Reader, m *domain.Message) (plain, htmlBody string, err error) {
	var found bodies
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return found.plain, found.html, nil
		}
		if err != nil && !message.IsUnknownCharset(err) {
			return "", "", err
		}
		if p == nil {
			continue
		}
		if err := readPart(p, &found, m); err != nil {
			return "", "", err
		}
	}
}

// readPart takes one part: a body goes to found, an attachment to the message.
func readPart(p *mail.Part, found *bodies, m *domain.Message) error {
	switch h := p.Header.(type) {
	case *mail.InlineHeader:
		kind, _, _ := h.ContentType()
		text, err := readLimited(p.Body, maxTextPart)
		if err != nil {
			return err
		}
		found.add(kind, text)
	case *mail.AttachmentHeader:
		kind, _, _ := h.ContentType()
		name, _ := h.Filename()
		size, err := io.Copy(ioutil.Discard, p.Body)
		if err != nil {
			return err
		}
		m.Attachments = append(m.Attachments, domain.Attachment{Name: name, MIMEType: kind, Size: int(size)})
	}
	return nil
}

// readLimited reads at most limit bytes and returns valid UTF-8.
func readLimited(r io.Reader, limit int64) (string, error) {
	b, err := ioutil.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		b = []byte(strings.ToValidUTF8(string(b), "�"))
	}
	return string(b), nil
}

// preview is the first words of a body on one line.
func preview(body string) string {
	flat := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(flat) <= previewLen {
		return flat
	}
	runes := []rune(flat)
	return strings.TrimSpace(string(runes[:previewLen])) + "…"
}
