package presentation

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"go-pocketbook-base/internal/installer/application"
	"go-pocketbook-base/internal/installer/domain"
)

type finder []domain.Reader

func (f finder) Find() []domain.Reader { return f }

type source []domain.Package

func (s source) Packages() ([]domain.Package, error) { return s, nil }

type deployer struct {
	fail map[string]error
	on   []string
}

func (d *deployer) Deploy(r domain.Reader, p domain.Package) (bool, error) {
	d.on = append(d.on, r.Label+"/"+p.ID)
	return true, d.fail[p.ID]
}

func console(input string, readers []domain.Reader, d *deployer, lang string) (Console, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return Console{
		In: strings.NewReader(input), Out: out, Tr: NewTranslator(lang), Version: "1.2.3",
		Install: application.Install{
			Finder: finder(readers), Source: source{{ID: "hello"}, {ID: "launcher"}}, Deployer: d,
		},
	}, out
}

func TestEveryTextIsTranslated(t *testing.T) {
	for _, lang := range []string{"pt-BR", "en"} {
		if m := NewTranslator("pt-BR").Missing(lang); len(m) != 0 {
			t.Errorf("%s lacks %v", lang, m)
		}
	}
}

func TestTheOnlyReaderIsUsedWithoutAskingAndTheWindowWaitsForEnter(t *testing.T) {
	d := &deployer{}
	c, out := console("\n", []domain.Reader{{Root: "F:\\", Label: "F:\\"}}, d, "pt-BR")
	if code := c.Run(""); code != ExitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	text := out.String()
	for _, want := range []string{"versão 1.2.3", "Leitor encontrado: F:\\", "[ok]   hello", "[ok]   launcher", "Pronto!", "Pressione Enter"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if len(d.on) != 2 {
		t.Errorf("both apps installed: %v", d.on)
	}
}

func TestWithoutAReaderTheInstructionsExplainWhatToDo(t *testing.T) {
	c, out := console("", nil, &deployer{}, "en")
	c.NoPause = true
	if code := c.Run(""); code != ExitFailed {
		t.Errorf("code %d", code)
	}
	if !strings.Contains(out.String(), "No PocketBook was found") || !strings.Contains(out.String(), "USB cable") {
		t.Errorf("\n%s", out)
	}
	if strings.Contains(out.String(), "Press Enter") {
		t.Error("NoPause does not wait")
	}
}

func TestSeveralReadersAskWhichOne(t *testing.T) {
	d := &deployer{}
	c, out := console("x\n9\n2\n\n", []domain.Reader{{Label: "A"}, {Label: "B"}}, d, "pt-BR")
	if code := c.Run(""); code != ExitOK || len(d.on) != 2 || d.on[0] != "B/hello" {
		t.Fatalf("the second reader is chosen: %v\n%s", d.on, out)
	}
	if strings.Count(out.String(), "Opção inválida") != 2 {
		t.Errorf("two invalid answers:\n%s", out)
	}

	c, _ = console("", []domain.Reader{{Label: "A"}, {Label: "B"}}, &deployer{}, "pt-BR")
	c.NoPause = true
	if code := c.Run(""); code != ExitFailed {
		t.Error("no answer, no installation")
	}
}

func TestAGivenDriveSkipsTheSearch(t *testing.T) {
	d := &deployer{}
	c, out := console("", nil, d, "pt-BR")
	c.NoPause = true
	if code := c.Run("G:\\"); code != ExitOK || len(d.on) != 2 || d.on[0] != "G:\\/hello" {
		t.Errorf("%v\n%s", d.on, out)
	}
	if strings.Contains(out.String(), "Procurando") {
		t.Error("nothing to search for")
	}
}

func TestAFailingAppIsReportedAndTheOthersStillInstall(t *testing.T) {
	d := &deployer{fail: map[string]error{"hello": errors.New("sem espaço")}}
	c, out := console("", []domain.Reader{{Label: "R"}}, d, "pt-BR")
	c.NoPause = true
	if code := c.Run(""); code != ExitFailed {
		t.Errorf("code %d", code)
	}
	if !strings.Contains(out.String(), "[erro] hello: sem espaço") || !strings.Contains(out.String(), "[ok]   launcher") ||
		!strings.Contains(out.String(), "Alguns aplicativos") {
		t.Errorf("\n%s", out)
	}
}

func TestAnInstallerWithoutAppsSaysSo(t *testing.T) {
	c, out := console("", []domain.Reader{{Label: "R"}}, &deployer{}, "en")
	c.NoPause = true
	c.Install.Source = source{}
	if code := c.Run(""); code != ExitFailed || !strings.Contains(out.String(), "Could not install") {
		t.Errorf("%d\n%s", code, out)
	}
}
