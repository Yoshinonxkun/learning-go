package main

import (
	"fmt"
	"os"
	"text/template"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("gogen benötigt zwei Argumente")
		fmt.Println("gogen [template] [typename]")
		os.Exit(1)
	}

	templateFileName := os.Args[1]
	typeName := os.Args[2]

	template, err := template.ParseFiles(templateFileName)
	if err != nil {
		fmt.Printf("Fehler beim Parsen: %v\n", err)
		os.Exit(1)
	}

	outName := fmt.Sprintf("gogen_%s_gen.go", typeName)
	fd, err := os.OpenFile(outName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Fehler beim Erzeugen des Zielfiles: %v\n", err)
		os.Exit(1)
	}
	defer fd.Close()

	data := struct {
		T string
	}{
		typeName,
	}
	template.Execute(fd, data)

}
