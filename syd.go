package main

import (
	"log"
	"os"

	"github.com/mibk/syd/core"
	"github.com/mibk/syd/ui/term"
)

func main() {
	log.SetPrefix("syd: ")
	log.SetFlags(0)

	ui := &term.UI{}
	if err := ui.Init(); err != nil {
		log.Fatalln("initializing ui:", err)
	}
	defer ui.Close()

	ed := core.NewEditor(ui)
	if len(os.Args) == 1 {
		ed.NewWindow()
	} else {
		for _, a := range os.Args[1:] {
			if err := ed.NewWindowFile(a); err != nil {
				panic(err)
			}
		}
	}
	ed.Main()
}

const (
	ModeNormal = iota
	ModeInsert
)
