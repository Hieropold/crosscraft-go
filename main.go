package main

import (
	"fmt"
	"net/http"
	"html/template"
	"database/sql"
	"math/rand"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Page struct {
	Title string
	Body  []byte
}

type Clue struct {
	cid int
	clue string
}

type Word struct {
	wid int
	word string
	clues[] Clue
}

var db *sql.DB

var indexTpl = template.Must(template.ParseFiles("templates/main.html", "templates/welcome.html"))
var quizTpl = template.Must(template.ParseFiles("templates/main.html", "templates/quiz.html"))

var totalWords int
var totalClues int
var random *rand.Rand

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := indexTpl.ExecuteTemplate(w, "main", nil);
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func quizHandler(w http.ResponseWriter, r *http.Request) {

	var err error

	// Load random word
	var randomWord Word
	randomWord, err = getRandomWord()
	if (err != nil) {
		panic(err.Error())
	}
	fmt.Printf("Random word: %s\n", randomWord.word)

	// Select random clue
	// Load 4 clues of other random words
	// Shuffle clues

	err = quizTpl.ExecuteTemplate(w, "main", nil)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getRandomOffset() (int, error) {
	var offset int

	stmt, err := db.Prepare("SELECT FLOOR(RAND() * COUNT(*)) AS offset FROM words")
	if err != nil {
		panic(err.Error())
	}
	defer stmt.Close()

	err = stmt.QueryRow().Scan(&offset)
	if (err != nil) {
		return 0, err
	}

	return offset, nil
}

func getWordByOffset(offset int) (Word, error) {
	var word Word

	err := db.QueryRow("SELECT wid, word FROM words LIMIT ?, 1", offset).Scan(&word.wid, &word.word)
	if (err != nil) {
		return word, err
	}

	rows, err := db.Query("SELECT cid, clue FROM clues WHERE wid = ?", word.wid)
	for rows.Next() {
		var clue Clue
		err = rows.Scan(&clue.cid, &clue.clue)
		if err != nil {
			return word, err
		}

		fmt.Printf("Clue: %s\n", clue.clue)
	}

	return word, nil

}

func getRandomWord() (Word, error) {
	var word Word

	offset, err := getRandomOffset()
	if (err != nil) {
		return word, err
	}

	word, err = getWordByOffset(offset)
	if (err != nil) {
		return word, err
	}

	return word, nil
}

func initCounts() {
	err := db.QueryRow("SELECT COUNT(*) AS total FROM words").Scan(&totalWords)
	if err != nil {
		panic(err.Error())
	}

	err = db.QueryRow("SELECT COUNT(*) AS total FROM clues").Scan(&totalClues)
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	fmt.Println("Crosscraft server starting")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/quiz", quizHandler)

	// Static assets
	http.Handle("/public", http.FileServer(http.Dir("./public/")))

	var err error
	db, err = sql.Open("mysql", "crosscraft:crosscraft@/crosscraft")
	if (err != nil) {
		panic(err.Error())
	}
	defer db.Close();

	err = db.Ping()
	if (err != nil) {
		panic(err.Error())
	}

	initCounts()
	fmt.Printf("Total words: %d\n", totalWords)
	fmt.Printf("Total clues: %d\n", totalClues)

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	fmt.Printf("Random: %d\n", random.Intn(totalWords))

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", nil)
}
