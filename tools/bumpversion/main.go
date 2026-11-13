// Command bumpversion increments the version stored in the .version file.
//
//	go run ./tools/bumpversion build   # every build:   1.2.3 -> 1.2.4
//	go run ./tools/bumpversion minor   # finished task: 1.2.4 -> 1.3.0
//	go run ./tools/bumpversion major   # big change:    1.3.0 -> 2.0.0
//
// It prints the new version, so build scripts can capture it.
package main

import (
	"flag"
	"fmt"
	"os"

	"go-pocketbook-base/internal/version"
)

func main() {
	file := flag.String("file", ".version", "version file")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bumpversion [-file .version] build|minor|major")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	v, err := version.BumpFile(*file, version.Kind(flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bumpversion:", err)
		os.Exit(1)
	}
	fmt.Println(v)
}
