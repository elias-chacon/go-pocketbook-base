package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"sync"

	"go-pocketbook-base/internal/bible/domain"
)

// localeFiles are where a PocketBook keeps its settings, the first one that names a language wins.
var localeFiles = []string{"/ebrmain/config/global.cfg", "/ebrmain/config/system.cfg"}

// posixVariables are the variables a Linux reads its language from.
var posixVariables = []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"}

// deviceLanguage is the language of the reader as the app understands it ("pt-BR" or "en"): what
// the reader says in its settings, then the POSIX variables, and the language of the shared settings
// of the apps (fallback) when none of them is known. Only Portuguese and English have translations.
func deviceLanguage(fallback string) string {
	for _, file := range localeFiles {
		if lang := fromConfig(file); lang != "" {
			return lang
		}
	}
	for _, key := range posixVariables {
		if lang := fromTag(os.Getenv(key)); lang != "" {
			return lang
		}
	}
	return fallback
}

// fromConfig reads a "language=..." line of a settings file ("" when there is none).
func fromConfig(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(strings.ToLower(line), "language=") {
			return fromTag(strings.Trim(line[len("language="):], `"' `))
		}
	}
	return ""
}

// fromTag maps a POSIX tag ("pt_BR.UTF-8") or a language name ("Portuguese") to a language of the
// app, "" when it is neither Portuguese nor English.
func fromTag(tag string) string {
	t := strings.ToLower(strings.TrimSpace(tag))
	switch {
	case strings.HasPrefix(t, "pt"), strings.HasPrefix(t, "portug"):
		return "pt-BR"
	case strings.HasPrefix(t, "en"):
		return "en"
	}
	return ""
}

// engines are the speech programs tried, with their arguments before the text.
var engines = [][]string{{"flite", "-t"}, {"espeak-ng"}, {"espeak"}}

// commandSpeaker reads aloud with a speech program of the system when there is one. The SDK of
// the reader has no speech API for apps, so on a reader without such a program Available is false
// and the app says so instead of pretending.
type commandSpeaker struct {
	mu   sync.Mutex
	args []string
	cmd  *exec.Cmd
}

// environSpeaker finds the speech program, if any.
func environSpeaker() domain.Speaker {
	for _, e := range engines {
		if path, err := exec.LookPath(e[0]); err == nil {
			return &commandSpeaker{args: append([]string{path}, e[1:]...)}
		}
	}
	return &commandSpeaker{}
}

// Available implements domain.Speaker.
func (s *commandSpeaker) Available() bool { return len(s.args) > 0 }

// Speak implements domain.Speaker: it starts the program and returns at once.
func (s *commandSpeaker) Speak(text string) error {
	s.Stop()
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd := exec.Command(s.args[0], append(append([]string(nil), s.args[1:]...), text)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	s.cmd = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

// Stop implements domain.Speaker.
func (s *commandSpeaker) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.cmd = nil
}
