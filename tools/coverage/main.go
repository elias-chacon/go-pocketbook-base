// Command coverage turns a Go coverage profile into the percentage that counts
// (the same exclusions as the Sonar quality gate) and, for the README badge, the
// JSON that shields.io's endpoint badge reads.
//
//	go test -coverpkg=./... -coverprofile=coverage.out ./...
//	go run ./tools/coverage                       # prints 95.3
//	go run ./tools/coverage -badge badge.json     # also writes the badge file
//	go run ./tools/coverage -min 80               # exit 1 below 80 %
//
// The continuous integration workflow publishes the badge file (see
// .github/workflows/ci.yml).
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

func main() {
	profile := flag.String("profile", "coverage.out", "the Go coverage profile")
	props := flag.String("sonar", "sonar-project.properties", "where the coverage exclusions are")
	badgePath := flag.String("badge", "", "write the shields.io endpoint JSON to this file")
	min := flag.Float64("min", 0, "exit with 1 when the coverage is below this percentage")
	flag.Parse()

	if err := run(*profile, *props, *badgePath, *min); err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
}

func run(profilePath, propsPath, badgePath string, min float64) error {
	pf, err := os.Open(profilePath)
	if err != nil {
		return err
	}
	defer pf.Close()
	var excluded []*regexp.Regexp
	if sf, err := os.Open(propsPath); err == nil {
		defer sf.Close()
		excluded = exclusions(sf)
	}
	res, err := measure(pf, excluded)
	if err != nil {
		return err
	}
	fmt.Printf("%.1f\n", res.Percent())
	if badgePath != "" {
		data, err := badgeJSON(res.Percent())
		if err != nil {
			return err
		}
		if err := os.WriteFile(badgePath, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	if res.Percent() < min {
		return fmt.Errorf("%.1f%% is below the minimum of %.1f%%", res.Percent(), min)
	}
	return nil
}
