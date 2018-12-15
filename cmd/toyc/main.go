// toyc is a toy compiler in Go.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/tools/go/packages"
)

func usage() {
	const use = `
Usage: toyc [OPTION]... [packages]
`
	fmt.Fprintln(os.Stderr, use[1:])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	// Pass command-line arguments uninterpreted to packages.Load so that it can
	// interpret them according to the conventions of the underlying build
	// system.
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		log.Fatalf("unable to load packages: %+v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	c := newCompiler()
	packages.Visit(pkgs, c.pre, c.post)
}
