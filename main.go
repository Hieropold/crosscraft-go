package main

import (
	"fmt"
	"net/http"
	"html/template"
)

type Page struct {
	Title string
	Body  []byte
}

var templates = template.Must(template.ParseFiles("templates/main.html", "templates/welcome.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "main", nil);
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func quizHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("quiz!!!");
}

func main() {
	fmt.Println("Crosscraft server starting")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/quiz", quizHandler)

	// Static assets
	http.Handle("/public", http.FileServer(http.Dir("./public/")))

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", nil)
}
