// Package presentation is the Installer's face: a console conversation meant for
// someone who double-clicked the program. It says what it does in plain words,
// asks only when it must, and waits for Enter at the end so the window does not
// vanish before the message can be read.
package presentation

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go-pocketbook-base/internal/installer/application"
	"go-pocketbook-base/internal/installer/domain"
	"go-pocketbook-base/internal/localization/i18n"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the installer's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Exit codes of Run.
const (
	ExitOK     = 0
	ExitFailed = 1
)

// Console is the conversation.
type Console struct {
	In      io.Reader
	Out     io.Writer
	Tr      *i18n.Bundle
	Version string
	Install application.Install
	// NoPause skips the final "press Enter" (scripts, tests).
	NoPause bool
}

// Run installs the apps and returns the exit code. A non-empty drive is the
// reader's root given by the person (skipping the search).
func (c Console) Run(drive string) int {
	in := bufio.NewReader(c.In)
	code := c.run(in, drive)
	if !c.NoPause {
		c.say("")
		c.say(c.Tr.T("press.enter"))
		in.ReadString('\n')
	}
	return code
}

func (c Console) run(in *bufio.Reader, drive string) int {
	c.say(c.Tr.T("title", c.Version))
	c.say("")

	reader, ok := c.pickReader(in, drive)
	if !ok {
		return ExitFailed
	}
	c.say(c.Tr.T("reader.using", reader.Label))
	c.say("")

	report, err := c.Install.Run(reader)
	if err != nil {
		c.say(c.Tr.T("error.generic", err.Error()))
		return ExitFailed
	}
	for _, o := range report.Outcomes {
		if o.Err != nil {
			c.say(c.Tr.T("app.failed", o.ID, o.Err.Error()))
		} else {
			c.say(c.Tr.T("app.ok", o.ID))
		}
	}
	c.say("")
	if report.Failed() > 0 {
		c.say(c.Tr.T("done.partial"))
		return ExitFailed
	}
	c.say(c.Tr.T("done"))
	return ExitOK
}

// pickReader is the reader to install on: the one given, the only one found, or
// the one the person chooses among several.
func (c Console) pickReader(in *bufio.Reader, drive string) (domain.Reader, bool) {
	if drive != "" {
		return domain.Reader{Root: drive, Label: drive}, true
	}
	c.say(c.Tr.T("searching"))
	readers := c.Install.Readers()
	switch len(readers) {
	case 0:
		c.say("")
		c.say(c.Tr.T("no.reader"))
		return domain.Reader{}, false
	case 1:
		return readers[0], true
	}
	c.say(c.Tr.T("choose"))
	for i, r := range readers {
		c.say(fmt.Sprintf("  %d) %s", i+1, r.Label))
	}
	for {
		fmt.Fprint(c.Out, "> ")
		line, err := in.ReadString('\n')
		n, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr == nil && n >= 1 && n <= len(readers) {
			return readers[n-1], true
		}
		if err != nil { // the input ended: nobody can answer
			return domain.Reader{}, false
		}
		c.say(c.Tr.T("choose.invalid"))
	}
}

func (c Console) say(text string) { fmt.Fprintln(c.Out, text) }
