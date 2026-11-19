package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "go-pocketbook-base/"

// moduleRoot is the repository root (this package is internal/architecture).
const moduleRoot = "../.."

// skippedDirs are not part of the source tree.
var skippedDirs = map[string]bool{".git": true, "dist": true, ".scannerwork": true}

// isSource reports whether path is a Go file that is not a test.
func isSource(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// moduleImports lists the imports of a Go file that belong to this module, as
// module-relative directories.
func moduleImports(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, imp := range file.Imports {
		// the standard library and third-party packages are not ours to police
		if p := strings.Trim(imp.Path.Value, `"`); strings.HasPrefix(p, modulePath) {
			dirs = append(dirs, strings.TrimPrefix(p, modulePath))
		}
	}
	return dirs, nil
}

// violationsIn returns what a source file imports that the rule does not allow,
// and how many module imports it checked.
func violationsIn(path string) (violations []string, checked int, err error) {
	rel, _ := filepath.Rel(moduleRoot, path)
	fromDir := filepath.ToSlash(filepath.Dir(rel))
	imports, err := moduleImports(path)
	for _, toDir := range imports {
		if !Allowed(fromDir, toDir) {
			violations = append(violations, rel+" imports "+toDir)
		}
	}
	return violations, len(imports), err
}

// TestEveryImportRespectsTheDependencyRule reads the imports of every non-test Go
// file of the repository and checks them against Rules.
func TestEveryImportRespectsTheDependencyRule(t *testing.T) {
	var violations []string
	checked := 0
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && skippedDirs[d.Name()]:
			return filepath.SkipDir
		case d.IsDir() || !isSource(path):
			return nil
		}
		v, n, err := violationsIn(path)
		violations, checked = append(violations, v...), checked+n
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 50 {
		t.Fatalf("only %d imports were checked: is the walk looking in the right place?", checked)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("breaks the dependency rule (see internal/architecture): %s", v)
	}
}

// TestEveryFolderOfInternalIsADeclaredContext makes a new top-level folder a
// decision: it has to be added to Contexts (and documented) on purpose.
func TestEveryFolderOfInternalIsADeclaredContext(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(moduleRoot, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, c := range Contexts {
		declared[c] = true
	}
	for _, e := range entries {
		if e.IsDir() && !declared[e.Name()] {
			t.Errorf("internal/%s is not a declared context: add it to architecture.Contexts and document it", e.Name())
		}
	}
	for c := range declared {
		if _, err := os.Stat(filepath.Join(moduleRoot, "internal", c)); err != nil {
			t.Errorf("the declared context %q does not exist", c)
		}
	}
}

func TestTheRuleItself(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		// the kernel depends on nothing
		{"internal/kernel/geom", "internal/kernel/version", false},
		{"internal/kernel/geom", "internal/toolkit/gfx", false},
		// the toolkit knows only the kernel
		{"internal/toolkit/widget", "internal/toolkit/gfx", true},
		{"internal/toolkit/widget", "internal/kernel/geom", true},
		{"internal/toolkit/widget", "internal/device/inkview", false},
		{"internal/toolkit/gfx", "internal/platform/domain", false},
		// the domain is pure: no toolkit, no device, no outer layer
		{"internal/platform/domain", "internal/localization/i18n", true},
		{"internal/platform/domain", "internal/toolkit/widget", false},
		{"internal/platform/domain", "internal/platform/infrastructure/jsonfiles", false},
		{"internal/platform/domain", "internal/platform/application", false},
		// the application uses the domain and the toolkit, never the device or the adapters
		{"internal/platform/application", "internal/platform/domain", true},
		{"internal/platform/application", "internal/toolkit/widget", true},
		{"internal/platform/application", "internal/device/inkview", false},
		{"internal/platform/application", "internal/platform/infrastructure/reader", false},
		// the infrastructure implements the ports
		{"internal/platform/infrastructure/reader", "internal/device/inkview", true},
		{"internal/platform/infrastructure/reader", "internal/platform/application", true},
		{"internal/platform/infrastructure/jsonfiles", "internal/platform/domain", true},
		{"internal/platform/infrastructure/jsonfiles", "internal/showcase", false},
		// the composition root wires the layers
		{"internal/platform", "internal/platform/infrastructure/reader", true},
		{"internal/platform", "internal/kernel/version", true},
		{"internal/platform", "internal/showcase", false},
		// the showcase is a screen: toolkit and languages only
		{"internal/showcase", "internal/toolkit/widget", true},
		{"internal/showcase", "internal/platform", false},
		{"internal/showcase", "internal/device/inkview", false},
		// examples teach the toolkit; they do not reach for the device
		{"examples/recipes", "internal/toolkit/widget", true},
		{"examples/components", "examples/harness", true},
		{"examples/components", "internal/device/inkview", false},
		// the Launcher: its own layers, the platform only for vocabulary
		{"internal/launcher/domain", "internal/platform/domain", true},
		{"internal/launcher/domain", "internal/toolkit/widget", false},
		{"internal/launcher/domain", "internal/launcher/infrastructure/processlauncher", false},
		{"internal/launcher/application", "internal/launcher/domain", true},
		{"internal/launcher/application", "internal/platform/infrastructure/process", false},
		{"internal/launcher/infrastructure/processlauncher", "internal/platform/infrastructure/process", true},
		{"internal/launcher/infrastructure/processlauncher", "internal/platform/infrastructure/reader", false},
		{"internal/launcher/presentation", "internal/toolkit/widget", true},
		{"internal/launcher/presentation", "internal/launcher/application", true},
		{"internal/launcher/presentation", "internal/launcher/infrastructure/processlauncher", false},
		{"internal/launcher/presentation", "internal/platform", false},
		{"internal/launcher/presentation", "internal/device/inkview", false},
		{"internal/installer/domain", "internal/platform/domain", false},
		{"internal/accounts/domain", "internal/platform/domain", false},
		{"internal/mail/domain", "internal/accounts/domain", false},
		{"internal/calendar/domain", "internal/mail/domain", false},
		{"internal/tasks/domain", "internal/mail/domain", false},
		{"internal/storage", "internal/mail/domain", false},
		{"internal/calendar/infrastructure/sqlstore", "internal/storage", true},
		{"internal/tasks/infrastructure/sqlstore", "internal/storage", true},
		{"internal/calendar/domain", "internal/storage", false},
		{"internal/tasks/domain", "internal/storage", false},
		{"internal/mail/infrastructure/sqlstore", "internal/storage", true},
		{"internal/mail/application", "internal/mail/domain", true},
		{"internal/mail/application", "internal/mail/infrastructure/graph", false},
		{"internal/mail/application", "internal/storage", false},
		{"internal/mail/infrastructure/graph", "internal/mail/infrastructure/mime", true},
		{"internal/mail/infrastructure/graph", "internal/mail/application", false},
		{"internal/mail/domain", "internal/storage", false},
		{"internal/tasks/domain", "internal/toolkit/widget", false},
		{"internal/calendar/infrastructure/icalendar", "internal/calendar/domain", true},
		{"internal/calendar/infrastructure/icalendar", "internal/toolkit/widget", false},
		{"internal/mail/infrastructure/mime", "internal/mail/domain", true},
		{"internal/mail/infrastructure/mime", "internal/toolkit/widget", false},
		{"internal/accounts/application", "internal/accounts/domain", true},
		{"internal/accounts/application", "internal/accounts/infrastructure/securefile", false},
		{"internal/accounts/infrastructure/securefile", "internal/accounts/domain", true},
		{"internal/accounts/infrastructure/securefile", "internal/accounts/application", false},
		{"internal/accounts/infrastructure/jsonaccounts", "internal/toolkit/widget", false},
		{"internal/launcher/infrastructure/github", "internal/platform/domain", true},
		{"internal/launcher/infrastructure/appinstaller", "internal/launcher/infrastructure/github", true},
		{"internal/launcher/infrastructure/appinstaller", "internal/platform", false},
		{"internal/launcher/infrastructure/iconcache", "internal/launcher/infrastructure/github", true},
		{"internal/launcher/infrastructure/prefsfile", "internal/launcher/domain", true},
		{"internal/launcher/infrastructure/prefsfile", "internal/toolkit/widget", false},
		{"internal/launcher/infrastructure/backgrounds", "internal/platform", false},
		{"internal/launcher/infrastructure/iconcache", "internal/toolkit/gfx", false},
		{"internal/installer/application", "internal/installer/domain", true},
		{"internal/installer/application", "internal/installer/infrastructure/drives", false},
		{"internal/installer/infrastructure/readerfs", "internal/installer/domain", true},
		{"internal/installer/infrastructure/readerfs", "internal/platform", false},
		{"internal/installer/presentation", "internal/installer/application", true},
		{"internal/installer/presentation", "internal/installer/infrastructure/readerfs", false},
		{"internal/installer/presentation", "internal/toolkit/widget", false},
		{"cmd/pbinstaller", "internal/platform", true},
		// nothing inside internal/ looks outwards
		{"internal/toolkit/widget", "examples/harness", false},
		{"internal/localization/i18n", "apps/hello", false},
		// composition roots and tools are free
		{"apps/hello", "internal/platform", true},
		{"apps/hello", "internal/device/inkview", true},
		{"tools/docgen", "examples/recipes", true},
		{"cmd/preview", "internal/showcase", true},
	}
	for _, c := range cases {
		if got := Allowed(c.from, c.to); got != c.want {
			t.Errorf("Allowed(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}

	if r, ok := RuleFor("internal/platform/infrastructure/reader"); !ok || r.From != "internal/platform/infrastructure" {
		t.Errorf("the most specific rule wins: %+v", r)
	}
	if _, ok := RuleFor("apps/hello"); ok {
		t.Error("apps are not limited")
	}
}
