package mime

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"

	"go-pocketbook-base/internal/mail/domain"
)

// NewMessageID makes a Message-ID (without the angle brackets) for a message sent
// from the address's domain.
func NewMessageID(from domain.Address) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system has no randomness: nothing safe to do
	}
	host := "localhost"
	if at := strings.LastIndex(from.Email, "@"); at >= 0 && at < len(from.Email)-1 {
		host = from.Email[at+1:]
	}
	return hex.EncodeToString(b) + "@" + host
}

// Compose writes the bytes of a draft: the headers, the text (UTF-8, quoted-
// printable) and the attachments. Bcc is not written: those recipients only get
// the message through the SMTP envelope (Draft.Recipients).
func Compose(d domain.Draft, now time.Time, messageID string) ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	var h mail.Header
	h.SetDate(now)
	h.SetMessageID(messageID)
	h.SetSubject(d.Subject)
	h.SetAddressList("From", toList([]domain.Address{d.From}))
	if len(d.To) > 0 {
		h.SetAddressList("To", toList(d.To))
	}
	if len(d.Cc) > 0 {
		h.SetAddressList("Cc", toList(d.Cc))
	}
	if d.InReplyTo != "" {
		h.SetMsgIDList("In-Reply-To", []string{trimBrackets(d.InReplyTo)})
	}
	if len(d.References) > 0 {
		ids := make([]string, len(d.References))
		for i, r := range d.References {
			ids[i] = trimBrackets(r)
		}
		h.SetMsgIDList("References", ids)
	}

	var out bytes.Buffer
	var err error
	if len(d.Attachments) == 0 {
		err = writeSingle(&out, h, d.Body)
	} else {
		err = writeWithAttachments(&out, h, d)
	}
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeSingle(out *bytes.Buffer, h mail.Header, body string) error {
	w, err := mail.CreateSingleInlineWriter(out, h)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return err
	}
	return w.Close()
}

func writeWithAttachments(out *bytes.Buffer, h mail.Header, d domain.Draft) error {
	mw, err := mail.CreateWriter(out, h)
	if err != nil {
		return err
	}
	text, err := mw.CreateInline()
	if err != nil {
		return err
	}
	var th mail.InlineHeader
	th.Set("Content-Type", "text/plain; charset=utf-8")
	part, err := text.CreatePart(th)
	if err != nil {
		return err
	}
	if _, err := part.Write([]byte(d.Body)); err != nil {
		return err
	}
	if err := part.Close(); err != nil {
		return err
	}
	if err := text.Close(); err != nil {
		return err
	}
	for _, a := range d.Attachments {
		var ah mail.AttachmentHeader
		ah.Set("Content-Type", contentType(a))
		ah.SetFilename(a.Name)
		w, err := mw.CreateAttachment(ah)
		if err != nil {
			return err
		}
		if _, err := w.Write(a.Data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}
	return mw.Close()
}

func contentType(a domain.DraftAttachment) string {
	if a.MIMEType != "" {
		return a.MIMEType
	}
	return "application/octet-stream"
}

func toList(list []domain.Address) []*mail.Address {
	out := make([]*mail.Address, len(list))
	for i, a := range list {
		out[i] = &mail.Address{Name: a.Name, Address: a.Email}
	}
	return out
}

func trimBrackets(id string) string {
	return strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(id), "<"), ">")
}
