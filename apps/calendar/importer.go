package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/calendar/infrastructure/icalendar"
)

// maxICS is the largest calendar read: the reader holds it in memory.
const maxICS = 8 << 20

// maxFound is how many .ics files are looked for.
const maxFound = 50

// icsImporter reads .ics files of the reader and iCalendar addresses into the agenda.
type icsImporter struct {
	agenda application.Agenda
	roots  []string
	web    *http.Client
}

// Files lists the .ics files of the reader's folders (two levels deep).
func (i icsImporter) Files() []string {
	var out []string
	for _, root := range i.roots {
		root := root
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				deep := strings.Count(strings.TrimPrefix(p, root), string(filepath.Separator)) > 2
				if deep || (p != root && strings.HasPrefix(info.Name(), ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.EqualFold(filepath.Ext(p), ".ics") && len(out) < maxFound {
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}

// Import reads a file path or an address and keeps its events.
func (i icsImporter) Import(source string) (int, []string, error) {
	r, err := i.open(source)
	if err != nil {
		return 0, nil, err
	}
	defer r.Close()
	events, warnings, err := icalendar.Parse(io.LimitReader(r, maxICS))
	if err != nil {
		return 0, warnings, err
	}
	n, err := i.agenda.Import(events)
	return n, warnings, err
}

func (i icsImporter) open(source string) (io.ReadCloser, error) {
	if strings.HasPrefix(strings.ToLower(source), "webcal://") {
		source = "https://" + source[len("webcal://"):]
	}
	lower := strings.ToLower(source)
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
		return os.Open(source)
	}
	client := i.web
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("could not reach the address: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("the server answered %s", resp.Status)
	}
	return resp.Body, nil
}
