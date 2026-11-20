package jsonfiles

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/platform/domain"
)

// updateFile is the name of the file that remembers the update checks.
const updateFile = "update.json"

// UpdateState is the domain.UpdateStateStore kept in a JSON file.
type UpdateState struct{ root string }

var _ domain.UpdateStateStore = UpdateState{}

// NewUpdateState creates the store of the reader whose storage is root.
func NewUpdateState(root string) UpdateState { return UpdateState{root: root} }

// Load implements domain.UpdateStateStore: a missing or broken file is a state with
// nothing remembered (the app then simply checks).
func (s UpdateState) Load() domain.UpdateState {
	var st domain.UpdateState
	data, err := os.ReadFile(filepath.Join(metaDir(s.root), updateFile))
	if err != nil || json.Unmarshal(data, &st) != nil {
		return domain.UpdateState{}
	}
	return st
}

// Save implements domain.UpdateStateStore.
func (s UpdateState) Save(st domain.UpdateState) error {
	return writeJSON(filepath.Join(metaDir(s.root), updateFile), st)
}
