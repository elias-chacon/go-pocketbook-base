package application

import (
	"errors"
	"strings"

	"go-pocketbook-base/internal/platform/domain"
)

// Uninstall is the use case "remove this app and everything it left on the reader": its icon on the
// home screen (the view.json entry and the icons), its file, its record, its log, the files its
// manifest lists as its data and the tables it has in the shared database. A file the manifest
// declares as shared (the accounts of the mail, tasks and calendar apps) goes only when no other
// installed app declares it too. Every step is tried even when an earlier one failed, and the
// failures are reported together: a half removed app must not be left because of one stubborn file.
type Uninstall struct {
	Apps    domain.AppRegistry
	Desktop domain.DesktopRemover
	Files   domain.AppFiles
	// EraseData drops what the app keeps in databases (nil: nothing to erase).
	EraseData func(appID string) error
}

// Run removes the app id.
func (u Uninstall) Run(id string) error {
	var gone domain.Manifest
	var others []domain.InstalledApp
	for _, a := range u.Apps.List() {
		if a.ID == id {
			gone = a.Manifest
		} else {
			others = append(others, a)
		}
	}
	files := append([]string{}, gone.Data...)
	for _, shared := range gone.Shares {
		if !sharedByAnother(others, shared) {
			files = append(files, shared)
		}
	}

	var problems []string
	note := func(err error) {
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	_, err := u.Desktop.Unregister(id)
	note(err)
	if u.EraseData != nil {
		note(u.EraseData(id))
	}
	note(u.Files.Remove(id, files))
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// sharedByAnother reports whether an installed app other than the one going declares file as shared.
func sharedByAnother(others []domain.InstalledApp, file string) bool {
	for _, a := range others {
		for _, s := range a.Shares {
			if s == file {
				return true
			}
		}
	}
	return false
}
