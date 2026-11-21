package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBump(t *testing.T) {
	cases := []struct {
		from string
		kind Kind
		want string
	}{
		{"1.0.0", Build, "1.0.1"},
		{"1.4.9", Build, "1.4.10"},
		{"1.4.9", Minor, "1.5.0"},
		{"1.4.9", Major, "2.0.0"},
	}
	for _, c := range cases {
		v, err := Parse(c.from)
		if err != nil {
			t.Fatal(err)
		}
		got, err := v.Bump(c.kind)
		if err != nil || got.String() != c.want {
			t.Errorf("%s bump %s = %v, %v; want %s", c.from, c.kind, got, err, c.want)
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, s := range []string{"", "1.0", "1.0.x", "1.-1.0", "1.0.0.0"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded", s)
		}
	}
}

func TestBumpUnknownKind(t *testing.T) {
	if _, err := (Version{1, 0, 0}).Bump("patch"); err == nil {
		t.Error("expected error")
	}
}

func TestBumpFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".version")
	if err := os.WriteFile(p, []byte("1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		kind Kind
		want string
	}{{Build, "1.0.1"}, {Build, "1.0.2"}, {Minor, "1.1.0"}} {
		v, err := BumpFile(p, step.kind)
		if err != nil || v.String() != step.want {
			t.Fatalf("BumpFile %s = %v, %v; want %s", step.kind, v, err, step.want)
		}
	}
	if data, _ := os.ReadFile(p); string(data) != "1.1.0\n" {
		t.Errorf("file = %q", data)
	}
}

func TestOlder(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"1.49.7", "1.49.8", true}, {"1.49.8", "1.49.8", false}, {"1.50.0", "1.49.9", false}, {"2.0.0", "1.99.99", false}, {"x", "1.0.0", false}, {"1.0.0", "dev", false}} {
		if got := Older(c.a, c.b); got != c.want {
			t.Errorf("Older(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
