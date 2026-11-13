// Command docsync keeps the code blocks of the documentation (docs/**/*.md) in
// sync with the tested samples in examples/:
//
//	go run ./tools/docsync          # refresh the blocks
//	go run ./tools/docsync -check   # only report: exit 1 when a page is stale
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "do not write, fail when a page is out of date")
	flag.Parse()

	stale, err := syncAll(*root, !*check)
	if err != nil {
		fmt.Fprintln(os.Stderr, "docsync:", err)
		os.Exit(2)
	}
	for _, p := range stale {
		fmt.Println(map[bool]string{true: "out of date: ", false: "updated: "}[*check] + p)
	}
	if *check && len(stale) > 0 {
		fmt.Fprintln(os.Stderr, "run `go run ./tools/docsync` to refresh the documentation")
		os.Exit(1)
	}
}
