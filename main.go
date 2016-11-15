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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/welcome.html");
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &Page{Title: "Welcome"}
	err = t.Execute(w, p);
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	fmt.Println("Crosscraft server starting")

	http.HandleFunc("/", indexHandler)

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", nil)
}
