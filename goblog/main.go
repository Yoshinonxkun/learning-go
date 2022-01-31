package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"
)

type Comment struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

type Page struct {
	Title      string
	LastChange time.Time
	Content    string
	Comments   []Comment
}

type Pages []Page

var (
	flagSrcFolder   = flag.String("src", "./seiten/", "blog folder")
	flagTmplFolder  = flag.String("tmpl", "./templates/", "template folder")
	flagFilesFolder = flag.String("files", "./files/", "path from the fileserver")
	flagPort        = flag.String("port", ":8001", "port of the webserver")
)

func loadPage(fpath string) (Page, error) {
	var p Page
	file, err := os.Stat(fpath)
	if err != nil {
		return p, fmt.Errorf("loadPage: %w", err)
	}

	p.Title = file.Name()
	p.LastChange = file.ModTime()
	p.Comments, err = loadComments(p.Title)
	if err != nil {
		return p, fmt.Errorf("loadPage.loadComments: %w", err)
	}

	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return p, fmt.Errorf("loadPage.ReadFile: %w", err)
	}

	p.Content = string(blackfriday.MarkdownCommon(b))

	return p, nil
}

func loadPages(src string) (Pages, error) {
	var ps Pages
	fs, err := ioutil.ReadDir(src)
	if err != nil {
		return ps, fmt.Errorf("loadPages.ReadDir: %w", err)
	}

	for _, f := range fs {
		if f.IsDir() {
			continue
		}

		fpath := filepath.Join(src, f.Name())
		p, err := loadPage(fpath)
		if err != nil {
			return ps, fmt.Errorf("loadPages.loadPage: %w", err)
		}
		ps = append(ps, p)
	}

	return ps, nil
}

func saveComments(title string, cs []Comment) error {
	fpath := filepath.Join("comments", title+".json")
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("saveComments: %w", err)
	}

	enc := json.NewEncoder(f)

	return enc.Encode(cs)
}

func loadComments(title string) ([]Comment, error) {
	var cs []Comment
	fpath := filepath.Join("comments", title+".json")
	f, err := os.Open(fpath)
	// Prüfung auf den Fehlerwert
	if errors.Is(err, os.ErrNotExist) {
		// kein Fehler, wenn Datei nicht existiert
		return cs, nil
	}
	// Alle anderen Fehler sind auch Fehler
	if err != nil {
		return cs, fmt.Errorf("loadComments: %w", err)
	}

	dec := json.NewDecoder(f)
	err = dec.Decode(&cs)

	return cs, err
}

func parseFiles(content string) (*template.Template, error) {
	return template.ParseFiles(
		filepath.Join(*flagTmplFolder, "base.tmpl.html"),
		filepath.Join(*flagTmplFolder, "header.tmpl.html"),
		filepath.Join(*flagTmplFolder, "footer.tmpl.html"),
		filepath.Join(*flagTmplFolder, "comment.tmpl.html"),
		filepath.Join(*flagTmplFolder, content))
}

func makePageHandlerFunc() http.HandlerFunc {
	tmpl, err := parseFiles("page.tmpl.html")
	if err != nil {
		fmt.Println(err)
		panic("makePageHandlerFunc: kann page.tmpl.html nicht parsen")
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		f := request.URL.Path[len("/page/"):]
		fpath := filepath.Join(*flagSrcFolder, f)
		p, err := loadPage(fpath)
		if err != nil {
			fmt.Println(err)
		}

		err = tmpl.ExecuteTemplate(writer, "base", p)
		if err != nil {
			fmt.Println("execute page template: ", err)
		}
	}
}

func makeCommentHandlerFunc() http.HandlerFunc {
	var mutex = &sync.Mutex{}

	return func(writer http.ResponseWriter, request *http.Request) {
		title := request.URL.Path[len("/comment/"):]

		// Formulardaten lesen
		name := request.FormValue("name")
		comment := request.FormValue("comment")

		// Kommentar erstellen
		c := Comment{Name: name, Comment: comment}

		mutex.Lock()
		cs, err := loadComments(title)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}

		cs = append(cs, c)
		err = saveComments(title, cs)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		mutex.Unlock()

		http.Redirect(writer, request, "/page/"+title, http.StatusFound)
	}
}

func makeAPIHandlerFunc() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ps, err := loadPages(*flagSrcFolder)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(writer)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", " ")

		err = enc.Encode(ps)
		if err != nil {
			fmt.Println("cannot encode pages to json")
		}
	}
}

func makeIndexHandlerFunc() http.HandlerFunc {
	tmpl, err := parseFiles("index.tmpl.html")
	if err != nil {
		fmt.Println(err)
		panic("makeIndexHandlerFunc: kann index.tmpl.html nicht parsen")
	}

	// periodically fetch index content
	var ps Pages
	go func() {
		for {
			ps, err = loadPages(*flagSrcFolder)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("index loaded")
			time.Sleep(30 * time.Second)
		}
	}()

	return func(writer http.ResponseWriter, request *http.Request) {
		err = tmpl.ExecuteTemplate(writer, "base", ps)
		if err != nil {
			fmt.Println("execute index template: ", err)
		}
	}
}

func main() {
	flag.Parse()

	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(*flagFilesFolder))))
	http.Handle("/api/", makeAPIHandlerFunc())
	http.HandleFunc("/page/", makePageHandlerFunc())
	http.HandleFunc("/comment/", makeCommentHandlerFunc())
	http.HandleFunc("/", makeIndexHandlerFunc())

	// Server starten
	fmt.Printf("Starte server auf Port %s\n", *flagPort)
	err := http.ListenAndServe(*flagPort, nil)
	if err != nil {
		fmt.Println("ListenAndServe: %w", err)
	}
}
