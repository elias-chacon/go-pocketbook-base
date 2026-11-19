package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var noExclusions []*regexp.Regexp

const profile = `mode: set
go-pocketbook-base/internal/a/a.go:1.1,2.2 4 1
go-pocketbook-base/internal/a/a.go:3.1,4.2 6 0
go-pocketbook-base/internal/a/a.go:1.1,2.2 4 0
go-pocketbook-base/cmd/tool/main.go:1.1,9.2 10 0
go-pocketbook-base/apps/hello/main.go:1.1,9.2 10 0
go-pocketbook-base/internal/device/inkview/x.go:1.1,9.2 10 0
go-pocketbook-base/internal/b/exec_linux.go:1.1,9.2 10 0
go-pocketbook-base/internal/b/b.go:1.1,2.2 10 3
`

func TestMeasureCountsEachBlockOnceAndCoveredWhenAnyRunRanIt(t *testing.T) {
	res, err := measure(strings.NewReader(profile), noExclusions)
	if err != nil {
		t.Fatal(err)
	}
	// a.go: 4 (covered by the first run, repeated uncovered) + 6 uncovered; the rest uncovered, b.go covered
	if res.Statements != 4+6+10+10+10+10+10 || res.Covered != 4+10 {
		t.Errorf("%+v", res)
	}
}

func TestTheSonarExclusionsLeaveOutWhatTheQualityGateLeavesOut(t *testing.T) {
	props := "sonar.projectKey=x\nsonar.coverage.exclusions=cmd/**,apps/*/main.go,**/exec_linux.go,internal/device/inkview/**,tools/*/main.go\n"
	res, err := measure(strings.NewReader(profile), exclusions(strings.NewReader(props)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Statements != 4+6+10 || res.Covered != 4+10 {
		t.Errorf("only a.go and b.go count: %+v", res)
	}
	if p := res.Percent(); p < 69.9 || p > 70.1 {
		t.Errorf("percent %.2f", p)
	}
}

func TestGlobsFollowSonarsRules(t *testing.T) {
	cases := []struct {
		glob, file string
		want       bool
	}{
		{"cmd/**", "cmd/x/main.go", true},
		{"cmd/**", "internal/cmd/x.go", false},
		{"apps/*/main.go", "apps/hello/main.go", true},
		{"apps/*/main.go", "apps/hello/sub/main.go", false},
		{"**/exec_linux.go", "exec_linux.go", true},
		{"**/exec_linux.go", "internal/p/exec_linux.go", true},
		{"**/exec_linux.go", "internal/p/exec_linux.go.bak", false},
		{"a.b", "axb", false},
	}
	for _, c := range cases {
		if got := globToRegexp(c.glob).MatchString(c.file); got != c.want {
			t.Errorf("%s vs %s = %v", c.glob, c.file, got)
		}
	}
}

func TestBrokenProfilesAreRejected(t *testing.T) {
	for _, bad := range []string{"nonsense", "a.go:1.1,2.2 x 1", "a.go:1.1,2.2 4 y", "a.go:1.1,2.2 4"} {
		if _, err := measure(strings.NewReader(bad), noExclusions); err == nil {
			t.Errorf("%q must be an error", bad)
		}
	}
	if res, err := measure(strings.NewReader("mode: set\n\n"), noExclusions); err != nil || res.Percent() != 0 {
		t.Errorf("an empty profile is 0 %%: %+v %v", res, err)
	}
}

func TestBadgeJSONHasTheShieldsShapeAndColors(t *testing.T) {
	data, err := badgeJSON(95)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"schemaVersion": 1`, `"label": "coverage"`, `"message": "95.0%"`, `"color": "brightgreen"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in %s", want, data)
		}
	}
	for percent, want := range map[float64]string{95: "brightgreen", 85: "green", 75: "yellowgreen", 65: "yellow", 55: "orange", 10: "red"} {
		if color(percent) != want {
			t.Errorf("%.0f -> %s", percent, color(percent))
		}
	}
}

func TestRunWritesTheBadgeAndEnforcesTheMinimum(t *testing.T) {
	dir := t.TempDir()
	prof := filepath.Join(dir, "coverage.out")
	os.WriteFile(prof, []byte(profile), 0o644)
	badge := filepath.Join(dir, "badge.json")

	if err := run(prof, filepath.Join(dir, "no-sonar.properties"), badge, 10); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(badge); !strings.Contains(string(data), "coverage") {
		t.Errorf("badge: %s", data)
	}
	if err := run(prof, filepath.Join(dir, "no-sonar.properties"), "", 99); err == nil {
		t.Error("below the minimum is an error")
	}
	if err := run(filepath.Join(dir, "missing.out"), "", "", 0); err == nil {
		t.Error("a missing profile is an error")
	}
	if err := run(prof, "", filepath.Join(dir, "no-such-dir", "b.json"), 0); err == nil {
		t.Error("an unwritable badge is an error")
	}
}
