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
	col := ed.NewColumn()
	if len(os.Args) == 1 {
		col.NewWindow()
	} else {
		for _, a := range os.Args[1:] {
			if _, err := col.NewWindowFile(a); err != nil {
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
