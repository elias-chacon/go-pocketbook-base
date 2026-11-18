package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/installer/domain"
)

type finder []domain.Reader

func (f finder) Find() []domain.Reader { return f }

type source struct {
	pkgs []domain.Package
	err  error
}

func (s source) Packages() ([]domain.Package, error) { return s.pkgs, s.err }

type deployer struct {
	fail map[string]error
	got  []string
}

func (d *deployer) Deploy(r domain.Reader, p domain.Package) (bool, error) {
	d.got = append(d.got, r.Root+":"+p.ID)
	return p.ID == "new", d.fail[p.ID]
}

func TestRunInstallsEveryPackageAndOneFailureDoesNotStopTheRest(t *testing.T) {
	boom := errors.New("disk full")
	d := &deployer{fail: map[string]error{"bad": boom}}
	uc := Install{
		Finder:   finder{{Root: "F:\\"}},
		Source:   source{pkgs: []domain.Package{{ID: "new"}, {ID: "bad"}, {ID: "old"}}},
		Deployer: d,
	}
	if len(uc.Readers()) != 1 {
		t.Fatal("one reader")
	}
	rep, err := uc.Run(uc.Readers()[0])
	if err != nil || len(rep.Outcomes) != 3 || rep.Failed() != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	if !rep.Outcomes[0].Registered || rep.Outcomes[2].Registered || rep.Outcomes[1].Err != boom {
		t.Errorf("outcomes: %+v", rep.Outcomes)
	}
	if len(d.got) != 3 || d.got[0] != "F:\\:new" {
		t.Errorf("deployed: %v", d.got)
	}
}

func TestRunReportsAMissingOrEmptyPayload(t *testing.T) {
	if _, err := (Install{Source: source{}, Deployer: &deployer{}}).Run(domain.Reader{}); err != domain.ErrNoPackages {
		t.Errorf("an installer without apps: %v", err)
	}
	broken := errors.New("unreadable")
	if _, err := (Install{Source: source{err: broken}}).Run(domain.Reader{}); err != broken {
		t.Errorf("a source error is returned: %v", err)
	}
}
