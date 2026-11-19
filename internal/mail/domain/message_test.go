package domain

import (
	"strings"
	"testing"
	"time"
)

var (
	me  = Address{Name: "Eu", Email: "eu@example.org"}
	ana = Address{Name: "Ana Souza", Email: "ana@example.org"}
	bia = Address{Email: "bia@example.org"}
)

func TestAddressesPrintAndParse(t *testing.T) {
	if ana.String() != `"Ana Souza" <ana@example.org>` || bia.String() != "bia@example.org" {
		t.Errorf("%s | %s", ana, bia)
	}
	if ana.Display() != "Ana Souza" || bia.Display() != "bia@example.org" || (Address{Name: "  ", Email: "x@y.z"}).Display() != "x@y.z" {
		t.Error("Display")
	}
	got, err := ParseAddress(`  Carlos Lima <carlos@example.org> `)
	if err != nil || got.Name != "Carlos Lima" || got.Email != "carlos@example.org" {
		t.Errorf("%+v %v", got, err)
	}
	if _, err := ParseAddress("not an address"); err == nil {
		t.Error("garbage")
	}
	if !ana.Same(Address{Email: "ANA@Example.org"}) || ana.Same(bia) {
		t.Error("Same ignores case")
	}
}

func TestFlagsAndUnread(t *testing.T) {
	m := Message{Flags: Seen | Flagged}
	if !m.Has(Seen) || !m.Has(Flagged) || m.Has(Answered) || m.Unread() {
		t.Error("flags")
	}
	if !(Message{}).Unread() {
		t.Error("a message without Seen is unread")
	}
}

func TestSubjectsGetOnePrefixWhateverCameBefore(t *testing.T) {
	cases := map[string]string{
		"Reunião": "Re: Reunião", "Re: Reunião": "Re: Reunião", "RE: re: Re:  Reunião": "Re: Reunião",
		"RES: Reunião": "Re: Reunião", "Aw: Reunião": "Re: Reunião", "": "Re:",
	}
	for in, want := range cases {
		if got := ReplySubject(in); got != want {
			t.Errorf("ReplySubject(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"Reunião": "Fwd: Reunião", "FW: Fwd: Reunião": "Fwd: Reunião", "ENC: Reunião": "Fwd: Reunião"} {
		if got := ForwardSubject(in); got != want {
			t.Errorf("ForwardSubject(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQuotingAddsOneLevelPerReply(t *testing.T) {
	if got := Quote("a\nb\r\n\nc\n"); got != "> a\n> b\n> \n> c" {
		t.Errorf("%q", got)
	}
	if got := Quote("> old\nnew"); got != ">> old\n> new" {
		t.Errorf("an old quote gets one more level: %q", got)
	}
}

func original() Message {
	return Message{
		MessageID: "<m1@example.org>", References: []string{"<m0@example.org>"},
		From: ana, To: []Address{me, bia}, Cc: []Address{{Email: "dir@example.org"}, me},
		Subject: "Re: Reunião", Body: "Podemos às 10h?",
	}
}

func TestReplyAnswersTheSenderAndKeepsTheThread(t *testing.T) {
	d := Reply(original(), me, false, "Ana escreveu:")
	if len(d.To) != 1 || !d.To[0].Same(ana) || len(d.Cc) != 0 {
		t.Errorf("only the sender: %+v %+v", d.To, d.Cc)
	}
	if d.Subject != "Re: Reunião" || d.InReplyTo != "<m1@example.org>" || len(d.References) != 2 || d.References[1] != "<m1@example.org>" {
		t.Errorf("the thread: %q %q %v", d.Subject, d.InReplyTo, d.References)
	}
	if !strings.Contains(d.Body, "Ana escreveu:\n> Podemos às 10h?") || !strings.HasPrefix(d.Body, "\n\n") {
		t.Errorf("the body leaves room to write above the quote: %q", d.Body)
	}
}

func TestReplyAllAnswersEveryoneButMe(t *testing.T) {
	d := Reply(original(), me, true, "x")
	emails := func(l []Address) string {
		var s []string
		for _, a := range l {
			s = append(s, a.Email)
		}
		return strings.Join(s, ",")
	}
	if emails(d.To) != "ana@example.org,bia@example.org" || emails(d.Cc) != "dir@example.org" {
		t.Errorf("to=%s cc=%s", emails(d.To), emails(d.Cc))
	}
}

func TestReplyGoesToTheReplyToAddressAndNeverToMe(t *testing.T) {
	m := original()
	m.ReplyTo = []Address{{Email: "list@example.org"}}
	if d := Reply(m, me, false, ""); len(d.To) != 1 || d.To[0].Email != "list@example.org" {
		t.Errorf("Reply-To wins: %+v", d.To)
	}
	mine := original()
	mine.From = me // answering my own message
	if d := Reply(mine, me, false, ""); len(d.To) != 0 {
		t.Errorf("never myself: %+v", d.To)
	}
	m = original()
	m.MessageID = ""
	if d := Reply(m, me, false, ""); d.InReplyTo != "" || len(d.References) != 1 {
		t.Errorf("without a Message-ID there is nothing to reference: %+v", d)
	}
}

func TestForwardCarriesTheOriginalTextUnderAHeader(t *testing.T) {
	d := Forward(original(), me, "---------- Forwarded ----------")
	if d.Subject != "Fwd: Re: Reunião" || len(d.To) != 0 || !strings.Contains(d.Body, "---------- Forwarded ----------\nPodemos às 10h?") {
		t.Errorf("%+v", d)
	}
}

func TestADraftNeedsASenderARecipientAndAReasonableSize(t *testing.T) {
	ok := Draft{From: me, To: []Address{ana}, Subject: "oi"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]Draft{
		"no sender":        {To: []Address{ana}},
		"no recipient":     {From: me},
		"a bad address":    {From: me, To: []Address{{Email: "not valid"}}},
		"two in one field": {From: me, To: []Address{{Email: "a@x.org,b@x.org"}}},
		"too big":          {From: me, To: []Address{ana}, Attachments: []DraftAttachment{{Name: "a", Data: make([]byte, MaxAttachmentBytes+1)}}},
	}
	for name, d := range cases {
		if d.Validate() == nil {
			t.Errorf("%s must be invalid", name)
		}
	}
	bccOnly := Draft{From: me, Bcc: []Address{ana}}
	if bccOnly.Validate() != nil {
		t.Error("a message to Bcc only can be sent")
	}
}

func TestRecipientsAreEveryoneOnceForTheEnvelope(t *testing.T) {
	d := Draft{To: []Address{ana, bia}, Cc: []Address{{Email: "ANA@example.org"}}, Bcc: []Address{bia, me, {}}}
	got := d.Recipients()
	if len(got) != 3 || got[2].Email != "eu@example.org" {
		t.Errorf("%+v", got)
	}
}

func TestTheDefaultAttributionNamesTheSenderAndTheDate(t *testing.T) {
	got := Attribution(ana, time.Date(2026, 1, 3, 9, 5, 0, 0, time.UTC))
	if got != "On 3 Jan 2026 09:05, Ana Souza wrote:" {
		t.Error(got)
	}
}

func TestParseAddressList(t *testing.T) {
	list, err := ParseAddressList("Ana <ana@example.org>, bia@example.org; ,cris@example.org")
	if err != nil || len(list) != 3 || list[0].Name != "Ana" || list[2].Email != "cris@example.org" {
		t.Errorf("list = %+v, %v", list, err)
	}
	if list, err := ParseAddressList("  "); err != nil || len(list) != 0 {
		t.Errorf("empty = %+v, %v", list, err)
	}
	if _, err := ParseAddressList("ok@example.org, não-é-endereço"); err == nil {
		t.Error("a bad address was accepted")
	}
}
