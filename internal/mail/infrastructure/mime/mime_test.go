package mime

import (
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/mail/domain"
)

func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

const plainMessage = `From: Ana Souza <ana@example.org>
To: Eu <eu@example.org>, bia@example.org
Cc: dir@example.org
Reply-To: Lista <lista@example.org>
Subject: Reuni=?UTF-8?Q?=C3=A3o?= de amanhã
Date: Sat, 03 Jan 2026 09:05:00 +0000
Message-ID: <m1@example.org>
In-Reply-To: <m0@example.org>
References: <m00@example.org> <m0@example.org>
Content-Type: text/plain; charset=utf-8

Podemos nos falar às 10h?

Abraço,
Ana
`

func TestAPlainMessageBecomesAMessage(t *testing.T) {
	m, err := Parse(strings.NewReader(crlf(plainMessage)))
	if err != nil {
		t.Fatal(err)
	}
	if m.From.Name != "Ana Souza" || m.From.Email != "ana@example.org" || len(m.To) != 2 || m.To[1].Email != "bia@example.org" || len(m.Cc) != 1 {
		t.Errorf("addresses: %+v %+v %+v", m.From, m.To, m.Cc)
	}
	if len(m.ReplyTo) != 1 || m.ReplyTo[0].Email != "lista@example.org" {
		t.Errorf("Reply-To: %+v", m.ReplyTo)
	}
	if m.Subject != "Reunião de amanhã" {
		t.Errorf("an encoded word is decoded: %q", m.Subject)
	}
	if m.Date.UTC() != time.Date(2026, 1, 3, 9, 5, 0, 0, time.UTC) {
		t.Errorf("date %v", m.Date)
	}
	if m.MessageID != "<m1@example.org>" || m.InReplyTo != "<m0@example.org>" || len(m.References) != 2 || m.References[1] != "<m0@example.org>" {
		t.Errorf("the thread: %q %q %v", m.MessageID, m.InReplyTo, m.References)
	}
	if !strings.HasPrefix(m.Body, "Podemos nos falar às 10h?") || !strings.HasSuffix(m.Body, "Ana") || strings.Contains(m.Body, "\r") {
		t.Errorf("body: %q", m.Body)
	}
	if m.Preview != "Podemos nos falar às 10h? Abraço, Ana" {
		t.Errorf("preview: %q", m.Preview)
	}
}

const alternative = `From: loja@example.org
To: eu@example.org
Subject: Oferta
Content-Type: multipart/alternative; boundary="B"

--B
Content-Type: text/html; charset=utf-8

<html><head><style>p{color:red}</style></head><body><p>Olá <b>Eu</b>,</p><ul><li>Item um</li><li>Item &amp; dois</li></ul><script>alert(1)</script></body></html>
--B--
`

func TestAnHTMLOnlyMessageIsShownAsText(t *testing.T) {
	m, err := Parse(strings.NewReader(crlf(alternative)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Body != "Olá Eu,\n\n• Item um\n• Item & dois" {
		t.Errorf("%q", m.Body)
	}
}

func TestThePlainPartWinsOverTheHTMLOne(t *testing.T) {
	raw := `From: a@example.org
Subject: x
Content-Type: multipart/alternative; boundary="B"

--B
Content-Type: text/plain; charset=utf-8

versão simples
--B
Content-Type: text/html; charset=utf-8

<p>versão <b>html</b></p>
--B--
`
	m, err := Parse(strings.NewReader(crlf(raw)))
	if err != nil || m.Body != "versão simples" {
		t.Errorf("%q %v", m.Body, err)
	}
}

func TestOldCharsetsAndQuotedPrintableAreDecoded(t *testing.T) {
	raw := "From: a@example.org\r\nSubject: =?ISO-8859-1?Q?Cora=E7=E3o?=\r\nContent-Type: text/plain; charset=iso-8859-1\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nCora=E7=E3o em =E9poca de caf=E9\r\n"
	m, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "Coração" || m.Body != "Coração em época de café" {
		t.Errorf("%q | %q", m.Subject, m.Body)
	}
}

func TestAnUnknownCharsetIsShownAsItIsAndABadHeaderDoesNotHideTheMessage(t *testing.T) {
	raw := "From: a@example.org\r\nDate: not a date\r\nContent-Type: text/plain; charset=x-klingon\r\n\r\nolá mundo\r\n"
	m, err := Parse(strings.NewReader(raw))
	if err != nil || !strings.Contains(m.Body, "mundo") || !m.Date.IsZero() {
		t.Errorf("%q %v %v", m.Body, m.Date, err)
	}
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Error("an empty input is not a message")
	}
}

func TestAttachmentsAreListedWithNameTypeAndSizeButNotLoaded(t *testing.T) {
	raw := `From: a@example.org
Subject: Contrato
Content-Type: multipart/mixed; boundary="M"

--M
Content-Type: text/plain; charset=utf-8

Segue o contrato.
--M
Content-Type: application/pdf; name="contrato.pdf"
Content-Disposition: attachment; filename="contrato.pdf"
Content-Transfer-Encoding: base64

SGVsbG8gUERG
--M--
`
	m, err := Parse(strings.NewReader(crlf(raw)))
	if err != nil || m.Body != "Segue o contrato." || len(m.Attachments) != 1 {
		t.Fatalf("%q %v %+v", m.Body, err, m.Attachments)
	}
	if a := m.Attachments[0]; a.Name != "contrato.pdf" || a.MIMEType != "application/pdf" || a.Size != len("Hello PDF") {
		t.Errorf("%+v", a)
	}
}

func TestALongBodyGetsAShortPreview(t *testing.T) {
	long := strings.Repeat("palavra ", 100)
	m, _ := Parse(strings.NewReader("From: a@example.org\r\n\r\n" + long))
	if !strings.HasSuffix(m.Preview, "…") || len([]rune(m.Preview)) > previewLen+1 || strings.Contains(m.Preview, "  ") {
		t.Errorf("%q", m.Preview)
	}
}

func draft() domain.Draft {
	return domain.Draft{
		From:    domain.Address{Name: "Eu Mesmo", Email: "eu@example.org"},
		To:      []domain.Address{{Name: "Ana Souza", Email: "ana@example.org"}},
		Cc:      []domain.Address{{Email: "dir@example.org"}},
		Bcc:     []domain.Address{{Email: "secreta@example.org"}},
		Subject: "Reunião às 10h", Body: "Olá, Ana!\n\nTudo bem? Até amanhã.\n",
		InReplyTo: "<m1@example.org>", References: []string{"<m0@example.org>", "<m1@example.org>"},
	}
}

func TestADraftSurvivesComposeAndParse(t *testing.T) {
	when := time.Date(2026, 1, 3, 9, 5, 0, 0, time.UTC)
	raw, err := Compose(draft(), when, "abc123@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secreta@example.org") {
		t.Error("Bcc never goes in the headers")
	}
	m, err := Parse(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "Reunião às 10h" || m.From.Name != "Eu Mesmo" || m.To[0].Email != "ana@example.org" || m.Cc[0].Email != "dir@example.org" {
		t.Errorf("%+v", m)
	}
	if m.MessageID != "<abc123@example.org>" || m.InReplyTo != "<m1@example.org>" || len(m.References) != 2 {
		t.Errorf("the thread: %q %q %v", m.MessageID, m.InReplyTo, m.References)
	}
	if m.Body != "Olá, Ana!\n\nTudo bem? Até amanhã." || m.Date.UTC() != when {
		t.Errorf("%q %v", m.Body, m.Date)
	}
}

func TestADraftWithAttachmentsSurvivesToo(t *testing.T) {
	d := draft()
	d.Attachments = []domain.DraftAttachment{
		{Name: "relatório.pdf", MIMEType: "application/pdf", Data: []byte("PDF-DATA-1234567890")},
		{Name: "notas.bin", Data: []byte{0, 1, 2, 3}},
	}
	raw, err := Compose(d, time.Now(), NewMessageID(d.From))
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(strings.NewReader(string(raw)))
	if err != nil || m.Body != "Olá, Ana!\n\nTudo bem? Até amanhã." || len(m.Attachments) != 2 {
		t.Fatalf("%q %v %+v", m.Body, err, m.Attachments)
	}
	if m.Attachments[0].Name != "relatório.pdf" || m.Attachments[0].Size != len("PDF-DATA-1234567890") || m.Attachments[1].MIMEType != "application/octet-stream" || m.Attachments[1].Size != 4 {
		t.Errorf("%+v", m.Attachments)
	}
}

func TestComposeRefusesADraftThatCannotBeSent(t *testing.T) {
	if _, err := Compose(domain.Draft{From: domain.Address{Email: "eu@example.org"}}, time.Now(), "id@x"); err == nil {
		t.Error("no recipient")
	}
}

func TestMessageIDsAreUniqueAndUseTheSendersDomain(t *testing.T) {
	a := NewMessageID(domain.Address{Email: "eu@example.org"})
	b := NewMessageID(domain.Address{Email: "eu@example.org"})
	if a == b || !strings.HasSuffix(a, "@example.org") || strings.ContainsAny(a, "<> ") {
		t.Errorf("%s %s", a, b)
	}
	if !strings.HasSuffix(NewMessageID(domain.Address{Email: "no-at"}), "@localhost") {
		t.Error("a sender without a domain still gets an id")
	}
}

func TestHTMLToTextRulesOfThumb(t *testing.T) {
	cases := map[string]string{
		"<p>um</p><p>dois</p>":                  "um\n\ndois",
		"a<br>b<br/>c":                          "a\nb\nc",
		"x &nbsp; y &lt;z&gt; &#169;":           "x y <z> ©",
		"<!-- c --><div>texto</div>":            "texto",
		"<h1>Título</h1>corpo":                  "Título\ncorpo",
		"<table><tr><td>a</td></tr></table>fim": "a\n\nfim",
		"   muitos     espaços   ":              "muitos espaços",
		"<p>a</p><p></p><p></p><p></p><p>b</p>": "a\n\nb",
		"":                                      "",
	}
	for in, want := range cases {
		if got := htmlToText(in); got != want {
			t.Errorf("htmlToText(%q) = %q, want %q", in, got, want)
		}
	}
}
