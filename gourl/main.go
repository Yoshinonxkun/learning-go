package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

var (
	flagOutput = flag.String("o", "", "output file")
	flagHeader = flag.Bool("header", false, "print HTTP-Header")
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("bitte nur eine url angeben")
		os.Exit(1)
	}
	url := args[0]
	if !validateURL(url) {
		fmt.Printf("nicht valide URL: %s\n", url)
		os.Exit(1)
	}

	var w io.Writer
	w = os.Stdout

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Fehler beim lesen von %s\n%#v\n", url, err)
	}
	defer resp.Body.Close()

	if *flagOutput != "" {
		err := os.MkdirAll(filepath.Dir(*flagOutput), 0755)
		if err != nil {
			fmt.Printf("Fehler beim Anlegen des Ordners: %v\n", err)
			os.Exit(1)
		}

		f, err := os.OpenFile(*flagOutput, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			fmt.Printf("Fehler beim Anlegen von %s\n%#v\n", *flagOutput, err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	if *flagHeader {
		for k, v := range resp.Header {
			fmt.Fprintf(w, "%s:\n", k)
			for i, l := range v {
				// Zeilen des Eintrags
				fmt.Fprintf(w, "  %03d: %s \n", i+1, l)
			}
		}
		fmt.Fprintf(w, "%#v", resp.Header)
		os.Exit(0)
	}

	io.Copy(w, resp.Body)
}

func validateURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return true
}
