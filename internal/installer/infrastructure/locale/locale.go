// Package locale finds out the language of the person running the installer.
package locale

import "strings"

// FromEnv reads the POSIX variables (LC_ALL, LC_MESSAGES, LANG, LANGUAGE) and
// returns a tag like "en-US", or "" when none is set to a real language.
func FromEnv(getenv func(string) string) string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		v := getenv(key)
		if i := strings.IndexAny(v, ".:@"); i >= 0 { // "pt_BR.UTF-8", "en:pt"
			v = v[:i]
		}
		if v != "" && v != "C" && v != "POSIX" {
			return strings.Replace(v, "_", "-", 1)
		}
	}
	return ""
}

// Detect is the language of this computer ("" when unknown: the caller picks its
// default). On Windows it asks the system; elsewhere it reads the environment.
func Detect(getenv func(string) string) string {
	if tag := fromSystem(); tag != "" {
		return tag
	}
	return FromEnv(getenv)
}
