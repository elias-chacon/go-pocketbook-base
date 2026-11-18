// Package application holds the use cases of the Installer: find the readers and
// install every app on one of them.
package application

import (
	"go-pocketbook-base/internal/installer/domain"
)

// Install is the use case "install the apps on the reader".
type Install struct {
	Finder   domain.ReaderFinder
	Source   domain.PackageSource
	Deployer domain.Deployer
}

// Report is the outcome of every package.
type Report struct{ Outcomes []domain.Outcome }

// Failed counts the packages that could not be installed.
func (r Report) Failed() int {
	n := 0
	for _, o := range r.Outcomes {
		if o.Err != nil {
			n++
		}
	}
	return n
}

// Readers are the PocketBooks connected now.
func (i Install) Readers() []domain.Reader { return i.Finder.Find() }

// Packages are the apps the installer carries.
func (i Install) Packages() ([]domain.Package, error) {
	pkgs, err := i.Source.Packages()
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, domain.ErrNoPackages
	}
	return pkgs, nil
}

// Run installs every package on r. One failing package does not stop the others:
// each has its own outcome.
func (i Install) Run(r domain.Reader) (Report, error) {
	pkgs, err := i.Packages()
	if err != nil {
		return Report{}, err
	}
	var rep Report
	for _, p := range pkgs {
		registered, err := i.Deployer.Deploy(r, p)
		rep.Outcomes = append(rep.Outcomes, domain.Outcome{ID: p.ID, Registered: registered, Err: err})
	}
	return rep, nil
}
