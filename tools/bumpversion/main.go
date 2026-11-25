// Command bumpversion increments a version file.
//
//	go run ./tools/bumpversion build   # every build:   1.2.3 -> 1.2.4   (.version, the release)
//	go run ./tools/bumpversion minor   # finished task: 1.2.4 -> 1.3.0
//	go run ./tools/bumpversion major   # big change:    1.3.0 -> 2.0.0
//
//	go run ./tools/bumpversion -app hello auto    # apps/hello/.version: BUILD+1 only if the app changed
//	go run ./tools/bumpversion -app hello check   # exit 1 when it changed and the version was not bumped
//	go run ./tools/bumpversion -app hello build   # BUILD+1 whatever happened (minor and major too)
//
// Every app has its own version, so a release that does not touch an app does not offer it as an
// update. It prints the version, so build scripts can capture it.
package main

import (
	"flag"
	"fmt"
	"os"

	"go-pocketbook-base/internal/kernel/version"
)

func main() {
	file := flag.String("file", ".version", "version file (without -app)")
	app := flag.String("app", "", "bump apps/<id>/.version instead")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bumpversion [-file .version] build|minor|major\n       bumpversion -app <id> auto|check|build|minor|major")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *app != "" {
		v, changed, err := bumpApp(".", *app, flag.Arg(0), sourcesOf)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bumpversion:", err)
			os.Exit(1)
		}
		if flag.Arg(0) == "check" && changed {
			fmt.Fprintf(os.Stderr, "bumpversion: apps/%s changed since %s: run `go run ./tools/bumpversion -app %s auto`\n", *app, v, *app)
			os.Exit(1)
		}
		fmt.Println(v)
		return
	}
	v, err := version.BumpFile(*file, version.Kind(flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bumpversion:", err)
		os.Exit(1)
	}
	fmt.Println(v)
}
