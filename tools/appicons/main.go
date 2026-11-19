package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	written, err := write("apps")
	if err != nil {
		fmt.Fprintln(os.Stderr, "appicons:", err)
		os.Exit(1)
	}
	sort.Strings(written)
	for _, id := range written {
		fmt.Printf("apps/%s/icon.png\n", id)
	}
}
