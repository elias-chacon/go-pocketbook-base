package locale

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestFromEnvReadsTheFirstRealLanguage(t *testing.T) {
	cases := []struct {
		env  map[string]string
		want string
	}{
		{nil, ""},
		{map[string]string{"LANG": "pt_BR.UTF-8"}, "pt-BR"},
		{map[string]string{"LANG": "pt_BR.UTF-8", "LC_ALL": "en_US.UTF-8"}, "en-US"},
		{map[string]string{"LC_ALL": "C", "LANG": "C.UTF-8"}, ""},
		{map[string]string{"LC_ALL": "POSIX", "LANGUAGE": "en:pt"}, "en"},
		{map[string]string{"LANG": "de_DE@euro"}, "de-DE"},
	}
	for _, c := range cases {
		if got := FromEnv(env(c.env)); got != c.want {
			t.Errorf("%v: %q, want %q", c.env, got, c.want)
		}
	}
}

func TestDetectAlwaysAnswers(t *testing.T) {
	// on Windows the system answers; elsewhere the environment does
	_ = Detect(env(map[string]string{"LANG": "en_US.UTF-8"}))
}
