package domain

// HomeScreen is the port to the reader's home screen (its desktop configuration).
type HomeScreen interface {
	// RemoveOrphans takes out of the home screen the entries of apps whose program is no longer on
	// the reader, and returns their ids. An app that is there is never touched. report, when not
	// nil, is told each id as its turn comes.
	RemoveOrphans(report func(id string)) ([]string, error)
}
