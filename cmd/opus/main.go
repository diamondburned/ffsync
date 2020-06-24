package main

import (
	"log"
	"os"

	"github.com/diamondburned/ffsync/opus"
)

func init() {
	log.SetFlags(0)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Usage:", os.Args[0], "file")
	}

	if err := opus.Convert(os.Args[1], opus.ConvertExt(os.Args[1], "opus")); err != nil {
		log.Fatalln(err)
	}
}
