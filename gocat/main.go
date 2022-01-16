package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Mindestens eine Datei als Parameter erwartet")
		os.Exit(1)
	}

	for _, fname := range os.Args[1:] {
		fmt.Println(fname)
		f, err := os.Open(fname)
		if err != nil {
			log.Printf(
				"Fehler beim Öffnen der Datei %s: %s",
				fname,
				err,
			)
			f.Close()
			continue
		}

		_, err = io.Copy(os.Stdout, f)
		if err != nil {
			log.Printf(
				"Fehler bei der Ausgabe von %s: %s",
				fname,
				err,
			)
		}
		f.Close()
	}
}
