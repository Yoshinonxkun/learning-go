package main

import (
	"flag"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	flagInFolder  = flag.String("in", "./", "Input-Ordner")
	flagOutFolder = flag.String("out", "", "Output-Folder")
	flagSize      = flag.String("size", "500x500", "maximale Größe")
)

type picSize struct {
	width  int
	height int
}

type resizeArgs struct {
	inPath  string
	outPath string
	size    picSize
}

type errorList struct {
	errs []error
}

func (e *errorList) add(err error) {
	if err != nil {
		e.errs = append(e.errs, err)
	}
}

func (e *errorList) hasError() bool {
	// liefert true ab einem vohandenen Fehler
	return len(e.errs) > 0
}

func (e *errorList) Error() string {
	if !e.hasError() {
		return ""
	}

	out := fmt.Sprintf("Number of errors %d: ", len(e.errs))
	for i, err := range e.errs {
		out = fmt.Sprintf("%s\n%d: %s",
			out,
			i,
			err.Error())
	}
	return out
}

func parseSize(s string) (picSize, error) {
	var ps picSize
	var err error

	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return ps, fmt.Errorf("%s nicht in der korrekten Form", s)
	}

	ps.width, err = strconv.Atoi(parts[0])
	if err != nil {
		return ps, fmt.Errorf("parseSize: ps.x: %w", err)
	}

	ps.height, err = strconv.Atoi(parts[1])
	if err != nil {
		return ps, fmt.Errorf("parseSize: ps.y: %w", err)
	}

	return ps, nil
}

func resize(ps picSize, r io.Reader, w io.Writer) error {
	img, format, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("fehler beim Decoding: %w", err)
	}
	if format != "jpeg" {
		return fmt.Errorf("nur jpeg wird unterstützt")
	}

	resized := imaging.Fit(
		img,
		ps.width, ps.height,
		imaging.Lanczos,
	)

	return jpeg.Encode(w, resized, nil)
}

func resizeClose(ps picSize, r io.ReadCloser, w io.WriteCloser) error {
	defer r.Close()
	defer w.Close()

	return resize(ps, r, w)
}

func useFile(filename string) bool {
	allowed := []string{".jpg", ".jpeg"}

	ext := filepath.Ext(filename)
	for _, e := range allowed {
		if strings.EqualFold(ext, e) {
			return true
		}
	}

	return false
}

func resizer(wg *sync.WaitGroup, c chan resizeArgs, errChan chan error) {
	for a := range c {
		log.Println("Verkleinere: ", a.inPath)
		inFile, err := os.Open(a.inPath)
		if err != nil {
			errChan <- fmt.Errorf("fehler beim Öffnen von %s: %w", a.inPath, err)
			continue
		}

		outFile, err := os.OpenFile(
			a.outPath,
			os.O_CREATE|os.O_WRONLY,
			07777)
		if err != nil {
			errChan <- fmt.Errorf("fehler beim Anlegen von %s: %w", a.outPath, err)
			inFile.Close()
			continue
		}

		err = resizeClose(a.size, inFile, outFile)
		if err != nil {
			errChan <- fmt.Errorf("fehler beim Verkleinern von %s: %w", a.inPath, err)
		}
	}
	wg.Done()
}

func resizeFolderImages(inFolder, outFolder string, size picSize) error {
	err := os.MkdirAll(outFolder, 0777)
	if err != nil {
		fmt.Println("Kann Zielverzeichnis nicht erzeugen: ", err)
	}

	dir, err := ioutil.ReadDir(inFolder)
	if err != nil {
		fmt.Println("Fehler beim Lesen des Ordners:")
		fmt.Println(err)
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	errList := &errorList{}
	errChan := make(chan error)
	resizeChan := make(chan resizeArgs)
	wg.Add(3)

	go resizer(wg, resizeChan, errChan)
	go resizer(wg, resizeChan, errChan)
	go resizer(wg, resizeChan, errChan)

	go func(errList *errorList, errChan chan error) {
		for err := range errChan {
			errList.add(err)
		}
	}(errList, errChan)

	for _, fi := range dir {
		if fi.IsDir() || !useFile(fi.Name()) {
			continue
		}

		inPath := filepath.Join(inFolder, fi.Name())
		outPath := filepath.Join(outFolder, fi.Name())

		resizeChan <- resizeArgs{inPath, outPath, size}
	}

	close(resizeChan)
	close(errChan)

	wg.Wait()

	if errList.hasError() {
		return errList
	}
	return nil
}

func main() {
	flag.Parse()

	size, err := parseSize(*flagSize)
	if err != nil {
		fmt.Println("Kann Größe nicht erzeugen: ", err)
		os.Exit(1)
	}

	outFolder := *flagSize
	if *flagOutFolder != "" {
		outFolder = *flagOutFolder
	}

	err = resizeFolderImages(*flagInFolder, outFolder, size)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
