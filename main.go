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
	Cid int
	Clue string
}

type Word struct {
	Wid int
	Word string
	Clues[] Clue
}

var db *sql.DB

var welcomeTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/welcome.html"))
var quizTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/quiz.html"))

var totalWords int
var totalClues int
var random *rand.Rand

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := welcomeTpl.ExecuteTemplate(w, "content", nil);
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

	clues := make([]Clue, 1)

	// Select random correct clue
	clues[0] = randomWord.Clues[getRandomOffset(len(randomWord.Clues))]

	// Load 4 random clues
	var randomClues []Clue
	randomClues, err = getRandomIncorrectClues(randomWord.Wid, 4)
	clues = append(clues, randomClues...)

	// Shuffle clues
	shuffled := make([]Clue, len(clues))
	perm := rand.Perm(len(clues))
	for i, v := range perm {
		shuffled[v] = clues[i]
	}

	for _, c := range shuffled {
		fmt.Printf("Clue: %s\n", c.Clue)
	}

	randomWord.Clues = shuffled

	err = quizTpl.ExecuteTemplate(w, "content", randomWord)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getRandomOffset(limit int) (int) {
	return random.Intn(limit)
}

func getWordByOffset(offset int) (Word, error) {
	var word Word

	err := db.QueryRow("SELECT wid, word FROM words LIMIT ?, 1", offset).Scan(&word.Wid, &word.Word)
	if (err != nil) {
		return word, err
	}

	rows, err := db.Query("SELECT cid, clue FROM clues WHERE wid = ?", word.Wid)
	for rows.Next() {
		var clue Clue
		err = rows.Scan(&clue.Cid, &clue.Clue)
		if err != nil {
			return word, err
		}
		word.Clues = append(word.Clues, clue)
	}

	return word, nil
}

func getRandomWord() (Word, error) {
	var word Word
	var err error

	offset := getRandomOffset(totalWords)

	word, err = getWordByOffset(offset)
	if (err != nil) {
		return word, err
	}

	return word, nil
}

func getRandomIncorrectClues(wid int, count int) ([]Clue, error) {
	var offset int
	randomClues := make([]Clue, count)

	for i := 0; i < count; i++ {
		offset = getRandomOffset(totalClues - 1)
		row := db.QueryRow("SELECT cid, clue FROM clues WHERE wid != ? LIMIT ?, 1", wid, offset)
		err := row.Scan(&randomClues[i].Cid, &randomClues[i].Clue)
		if err != nil {
			return nil, err
		}
	}

	return randomClues, nil
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
	fmt.Printf("Random generator is initialized\n");

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", nil)
}
