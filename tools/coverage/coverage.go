package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// module is the Go module path: coverage profiles name files by it.
const module = "go-pocketbook-base/"

// Result is the coverage of the code that counts.
type Result struct {
	Statements, Covered int
}

// Percent is covered/statements as a percentage (0 when there is nothing to cover).
func (r Result) Percent() float64 {
	if r.Statements == 0 {
		return 0
	}
	return 100 * float64(r.Covered) / float64(r.Statements)
}

// globToRegexp turns the Sonar-style patterns of sonar.coverage.exclusions
// (`**` crosses folders, `*` does not) into a regular expression.
func globToRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(glob[i:], "**"):
			b.WriteString(".*")
			i++
		case glob[i] == '*':
			b.WriteString("[^/]*")
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// exclusions reads sonar.coverage.exclusions from a sonar-project.properties, so
// the badge counts exactly what the quality gate counts.
func exclusions(properties io.Reader) []*regexp.Regexp {
	const key = "sonar.coverage.exclusions="
	var out []*regexp.Regexp
	sc := bufio.NewScanner(properties)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, key) {
			continue
		}
		for _, g := range strings.Split(strings.TrimPrefix(line, key), ",") {
			if g = strings.TrimSpace(g); g != "" {
				out = append(out, globToRegexp(g))
			}
		}
	}
	return out
}

// entry is one line of a profile: path.go:10.2,12.5 3 1.
type entry struct {
	file, key    string
	stmts, count int
}

// parseEntry reads a profile line (the module prefix is removed from the file).
func parseEntry(line string) (entry, error) {
	colon := strings.LastIndex(line, ":")
	if colon < 0 {
		return entry{}, fmt.Errorf("coverage: cannot read %q", line)
	}
	fields := strings.Fields(line[colon+1:])
	if len(fields) != 3 {
		return entry{}, fmt.Errorf("coverage: cannot read %q", line)
	}
	stmts, err1 := strconv.Atoi(fields[1])
	count, err2 := strconv.Atoi(fields[2])
	if err1 != nil || err2 != nil {
		return entry{}, fmt.Errorf("coverage: cannot read %q", line)
	}
	return entry{
		file: strings.TrimPrefix(line[:colon], module), key: line[:colon] + ":" + fields[0],
		stmts: stmts, count: count,
	}, nil
}

// measure reads a Go coverage profile. A block that appears more than once (one
// test binary per package, each reporting every package) counts once and is
// covered when any of them ran it. Files matching an exclusion are left out.
func measure(profile io.Reader, excluded []*regexp.Regexp) (Result, error) {
	type block struct {
		stmts   int
		covered bool
	}
	blocks := map[string]*block{}
	sc := bufio.NewScanner(profile)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}
		e, err := parseEntry(line)
		if err != nil {
			return Result{}, err
		}
		if skip(e.file, excluded) {
			continue
		}
		b := blocks[e.key]
		if b == nil {
			b = &block{stmts: e.stmts}
			blocks[e.key] = b
		}
		b.covered = b.covered || e.count > 0
	}
	var res Result
	for _, b := range blocks {
		res.Statements += b.stmts
		if b.covered {
			res.Covered += b.stmts
		}
	}
	return res, sc.Err()
}

func skip(file string, excluded []*regexp.Regexp) bool {
	for _, re := range excluded {
		if re.MatchString(file) {
			return true
		}
	}
	return false
}

// badge is the JSON that shields.io's endpoint badge reads.
type badge struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// color follows the usual thresholds.
func color(percent float64) string {
	switch {
	case percent >= 90:
		return "brightgreen"
	case percent >= 80:
		return "green"
	case percent >= 70:
		return "yellowgreen"
	case percent >= 60:
		return "yellow"
	case percent >= 50:
		return "orange"
	}
	return "red"
}

// badgeJSON is the shields.io endpoint answer for a coverage percentage.
func badgeJSON(percent float64) ([]byte, error) {
	return json.MarshalIndent(badge{
		SchemaVersion: 1, Label: "coverage", Message: fmt.Sprintf("%.1f%%", percent), Color: color(percent),
	}, "", "  ")
}
