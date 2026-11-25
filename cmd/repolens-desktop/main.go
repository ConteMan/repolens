//go:build darwin || windows

package main

import (
	"flag"
	"log"

	"github.com/ConteMan/repolens/internal/desktop"
)

func main() {
	language := flag.String("lang", "", "Interface language: zh or en (default follows the system)")
	flag.Parse()
	if flag.NArg() > 1 || (*language != "" && *language != "zh" && *language != "en") {
		log.Fatal("usage: repolens-desktop [-lang zh|en] [repository directory]")
	}
	if err := desktop.Run(flag.Arg(0), *language); err != nil {
		log.Fatal(err)
	}
}
