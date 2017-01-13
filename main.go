package main

import (
	"fmt"
	"net/http"
	"html/template"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Page struct {
	Title string
	Body  []byte
}

var db *sql.DB

var indexTpl = template.Must(template.ParseFiles("templates/main.html", "templates/welcome.html"))
var quizTpl = template.Must(template.ParseFiles("templates/main.html", "templates/quiz.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := indexTpl.ExecuteTemplate(w, "main", nil);
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func quizHandler(w http.ResponseWriter, r *http.Request) {
	err := quizTpl.ExecuteTemplate(w, "main", nil)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load random 5
}

func getRandomOffset() (int, error) {
	var offset int
	err := db.QueryRow("SELECT FLOOR(RAND() * COUNT(*)) AS offset FROM words").Scan(&offset)
	if (err != nil) {
		return 0, err
	}

	return 0, nil
}

func getWordByOffset(offset int) (string, error) {
	var word string
	err := db.QueryRow("SELECT word FROM words LIMIT ?, 1", offset).Scan(&word)
	if (err != nil) {
		return "", err
	}

	return word, nil

}

func getRandomWord() (string, error) {
	offset, err := getRandomOffset()
	if (err != nil) {
		return "", err
	}

	word, err := getWordByOffset(offset)
	if (err != nil) {
		return "", err
	}

	return word, nil
}

func main() {
	fmt.Println("Crosscraft server starting")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/quiz", quizHandler)

	// Static assets
	http.Handle("/public", http.FileServer(http.Dir("./public/")))

	db, err := sql.Open("mysql", "crosscraft:crosscraft@/crosscraft")
	if (err != nil) {
		panic(err.Error())
	}
	defer db.Close();

	err = db.Ping()
	if (err != nil) {
		panic(err.Error())
	}

	var randomWord string
	randomWord, err = getRandomWord()
	if (err != nil) {
		panic(err.Error())
	}
	fmt.Printf("Random word: %s", randomWord)

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", nil)
}
