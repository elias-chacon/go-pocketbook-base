package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MaxAttachmentBytes bounds what a draft carries: the reader holds the message in
// memory while it builds and sends it.
const MaxAttachmentBytes = 20 << 20

// DraftAttachment is a file to send.
type DraftAttachment struct {
	Name     string
	MIMEType string
	Data     []byte
}

// Draft is a message being written.
type Draft struct {
	ID          string // set when the draft is kept on the reader
	AccountID   string
	From        Address
	To, Cc, Bcc []Address
	Subject     string
	Body        string // plain text
	InReplyTo   string // the Message-ID being answered
	References  []string
	Attachments []DraftAttachment
}

// Validate checks that the draft can be sent.
func (d Draft) Validate() error {
	var problems []string
	if d.From.Email == "" {
		problems = append(problems, "there is no sender")
	}
	if len(d.To)+len(d.Cc)+len(d.Bcc) == 0 {
		problems = append(problems, "there is no recipient")
	}
	for _, list := range [][]Address{d.To, d.Cc, d.Bcc} {
		for _, a := range list {
			if _, err := ParseAddress(a.Email); err != nil || strings.ContainsAny(a.Email, " ,;<>") {
				problems = append(problems, fmt.Sprintf("%q is not an address", a.Email))
			}
		}
	}
	total := 0
	for _, a := range d.Attachments {
		total += len(a.Data)
	}
	if total > MaxAttachmentBytes {
		problems = append(problems, fmt.Sprintf("the attachments are larger than %d MB", MaxAttachmentBytes>>20))
	}
	if len(problems) > 0 {
		return errors.New("mail: " + strings.Join(problems, "; "))
	}
	return nil
}

// Recipients are everyone the draft is delivered to (To, Cc and Bcc), without
// repeats: what goes in the SMTP envelope.
func (d Draft) Recipients() []Address { return dedupe(d.To, d.Cc, d.Bcc) }

func dedupe(lists ...[]Address) []Address {
	var out []Address
	for _, list := range lists {
		for _, a := range list {
			if a.Email != "" && !containsAddress(out, a) {
				out = append(out, a)
			}
		}
	}
	return out
}

func containsAddress(list []Address, a Address) bool {
	for _, b := range list {
		if a.Same(b) {
			return true
		}
	}
	return false
}

var replyPrefix = regexp.MustCompile(`(?i)^\s*(re|res|aw|sv)\s*:\s*`)
var forwardPrefix = regexp.MustCompile(`(?i)^\s*(fwd?|enc|wg|vs)\s*:\s*`)

// ReplySubject is the subject of an answer: one "Re:", however many came before.
func ReplySubject(subject string) string {
	return strings.TrimSpace("Re: " + strip(replyPrefix, subject))
}

// ForwardSubject is the subject of a forward: one "Fwd:".
func ForwardSubject(subject string) string {
	return strings.TrimSpace("Fwd: " + strip(forwardPrefix, subject))
}

func strip(re *regexp.Regexp, s string) string {
	for re.MatchString(s) {
		s = re.ReplaceAllString(s, "")
	}
	return strings.TrimSpace(s)
}

// Quote prefixes every line of text with "> ", the way a reply quotes the message.
func Quote(text string) string {
	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, ">") {
			lines[i] = ">" + l // already quoted: one more level, without a space
		} else {
			lines[i] = "> " + l
		}
	}
	return strings.Join(lines, "\n")
}

// Reply starts an answer to a message. me is the account's own address (never a
// recipient of the answer). all answers everyone. attribution is the line that
// introduces the quote ("On 3 Jan, Ana wrote:"), already in the person's language.
func Reply(orig Message, me Address, all bool, attribution string) Draft {
	to := orig.ReplyTo
	if len(to) == 0 {
		to = []Address{orig.From}
	}
	d := Draft{
		From: me, Subject: ReplySubject(orig.Subject), InReplyTo: orig.MessageID,
		References: append(append([]string(nil), orig.References...), orig.MessageID),
		Body:       "\n\n" + attribution + "\n" + Quote(orig.Body) + "\n",
	}
	if orig.MessageID == "" {
		d.References = append([]string(nil), orig.References...)
	}
	d.To = without(dedupe(to), me)
	if all {
		d.To = without(dedupe(to, orig.To), me)
		d.Cc = without(dedupe(orig.Cc), me)
		d.Cc = without(d.Cc, d.To...)
	}
	return d
}

// Forward starts a forward of a message. header is the block that introduces the
// original ("---------- Forwarded message ----------" and its fields), already in
// the person's language. Attachments are not copied: they would have to be
// downloaded first.
func Forward(orig Message, me Address, header string) Draft {
	return Draft{
		From: me, Subject: ForwardSubject(orig.Subject),
		Body: "\n\n" + header + "\n" + orig.Body + "\n",
	}
}

// without returns list minus the addresses in drop.
func without(list []Address, drop ...Address) []Address {
	var out []Address
	for _, a := range list {
		if !containsAddress(drop, a) {
			out = append(out, a)
		}
	}
	return out
}

// Attribution is the default English line that introduces a quote.
func Attribution(from Address, when time.Time) string {
	return fmt.Sprintf("On %s, %s wrote:", when.Format("2 Jan 2006 15:04"), from.Display())
}
