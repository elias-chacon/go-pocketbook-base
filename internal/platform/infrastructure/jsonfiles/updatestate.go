package jsonfiles

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/platform/domain"
)

// updateDir is the folder of .pbgo that holds one update file per app: what an app remembers about
// its own checks (when it asked, the build it found, the one it was told "later" to) is not another
// app's, so each has <id>.json here.
const updateDir = "update"

// UpdateState is the domain.UpdateStateStore of one app, kept in a JSON file.
type UpdateState struct{ root, appID string }

var _ domain.UpdateStateStore = UpdateState{}

// UpdatePath is the file where the app appID remembers its update checks.
func UpdatePath(root, appID string) string {
	return filepath.Join(metaDir(root), updateDir, appID+".json")
}

// NewUpdateState creates the store of the app appID of the reader whose storage is root.
func NewUpdateState(root, appID string) UpdateState { return UpdateState{root: root, appID: appID} }

// Load implements domain.UpdateStateStore: a missing or broken file is a state with
// nothing remembered (the app then simply checks).
func (s UpdateState) Load() domain.UpdateState {
	var st domain.UpdateState
	data, err := os.ReadFile(UpdatePath(s.root, s.appID))
	if err != nil || json.Unmarshal(data, &st) != nil {
		return domain.UpdateState{}
	}
	return st
}

// Save implements domain.UpdateStateStore.
func (s UpdateState) Save(st domain.UpdateState) error {
	return writeJSON(UpdatePath(s.root, s.appID), st)
}
