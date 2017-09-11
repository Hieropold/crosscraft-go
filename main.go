package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"recaptcha"
	"session"
	"strconv"
	"time"
)

type Clue struct {
	Cid  int
	Clue string
}

type Word struct {
	Wid   int
	Word  string
	Clues []Clue
}

type QuizPage struct {
	Word     Word
	Score    int
	MaxScore int
}

type DBConfig struct {
	User     string
	Password string
	DBName   string
	Host     string
}

func (c DBConfig) ConnString(maskPass bool) string {
	if maskPass {
		return c.User + ":<pass>@tcp(" + c.Host + ")/" + c.DBName
	}

	return c.User + ":" + c.Password + "@tcp(" + c.Host + ")/" + c.DBName
}

var db *sql.DB

var welcomeTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/welcome.html"))
var quizTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/quiz.html"))
var successTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/success.html"))
var failTpl = template.Must(template.ParseFiles("templates/header.html", "templates/footer.html", "templates/fail.html"))

var totalWords int
var totalClues int
var random *rand.Rand

func checkAccess(s session.Session) (bool) {
	return s.IsVerified()
}

func logWrapper(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h(w, r)
		duration := time.Now().Sub(startTime)
		fmt.Printf("Request %s: %d ns\n", r.URL, duration)
	}
}

func sessionLoader(h func(http.ResponseWriter, *http.Request, session.Session)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := session.Start(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h(w, r, s)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request, s session.Session) {
	var err error

	type WelcomeData struct {
		IsVerified   bool
		RecaptchaKey string
	}
	var data WelcomeData
	data.IsVerified = s.IsVerified()
	data.RecaptchaKey = recaptcha.Key
	err = welcomeTpl.ExecuteTemplate(w, "content", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func verifyHumanHandler(w http.ResponseWriter, r *http.Request, s session.Session) {
	if r.Method != "POST" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	r.ParseForm()

	token := r.Form["g-recaptcha-response"][0]

	success, err := recaptcha.Verify(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if success {
		s.SetVerified()
		s.Save(w, r)
		http.Redirect(w, r, "/quiz", http.StatusTemporaryRedirect)
		return
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
}

func quizHandler(w http.ResponseWriter, r *http.Request, s session.Session) {

	var err error

	granted := checkAccess(s)
	if !granted {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Load current user score
	score, max := s.GetScore()

	// Load random word
	var randomWord Word
	randomWord, err = getRandomWord()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

	randomWord.Clues = shuffled

	var quizPage QuizPage = QuizPage{
		Word:     randomWord,
		Score:    score,
		MaxScore: max,
	}

	err = quizTpl.ExecuteTemplate(w, "content", quizPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func answerHandler(w http.ResponseWriter, r *http.Request, s session.Session) {
	granted := checkAccess(s)
	if !granted {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	params := r.URL.Query()

	if params["wid"] == nil || params["cid"] == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var err error
	var wid int
	var cid int

	wid, err = strconv.Atoi(params["wid"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cid, err = strconv.Atoi(params["cid"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var isCorrect bool
	isCorrect, err = isCorrectClue(wid, cid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isCorrect {
		s.IncreaseScore()
		s.Save(w, r)
		err = successTpl.ExecuteTemplate(w, "content", nil)
	} else {
		s.ResetScore()
		s.Save(w, r)
		err = failTpl.ExecuteTemplate(w, "content", nil)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check DB connection
	err := db.Ping()
	if err != nil {
		fmt.Printf("Healthcheck error: %s", err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("I'm OK"))
}

func getRandomOffset(limit int) int {
	return random.Intn(limit)
}

func isCorrectClue(wid int, cid int) (bool, error) {
	var rawExists bool
	err := db.QueryRow("SELECT 1 FROM clues WHERE wid = ? AND cid = ?", wid, cid).Scan(&rawExists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func getWordByOffset(offset int) (Word, error) {
	var word Word

	err := db.QueryRow("SELECT wid, word FROM words LIMIT ?, 1", offset).Scan(&word.Wid, &word.Word)
	if err != nil {
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
	if err != nil {
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

	recaptcha.Key = os.Getenv("CROSSCRAFT_RECAPTCHA_KEY")
	if recaptcha.Key == "" {
		fmt.Printf("Missing reCaptcha token\n")
		return
	}
	recaptcha.Secret = os.Getenv("CROSSCRAFT_RECAPTCHA_SECRET")
	if recaptcha.Secret == "" {
		fmt.Printf("Missing reCaptcha secret\n")
		return
	}
	fmt.Printf("reCaptcha token: %s\n", recaptcha.Key)
	fmt.Printf("reCaptcha secret: %s\n", recaptcha.Secret)

	var dbCfg DBConfig
	dbCfg.User = os.Getenv("CROSSCRAFT_DB_USER")
	dbCfg.Password = os.Getenv("CROSSCRAFT_DB_PASSWORD")
	dbCfg.DBName = os.Getenv("CROSSCRAFT_DB_NAME")
	dbCfg.Host = os.Getenv("CROSSCRAFT_DB_HOST")
	fmt.Printf("DB connection string: %s\n", dbCfg.ConnString(true))

	http.HandleFunc("/", logWrapper(sessionLoader(indexHandler)))
	http.HandleFunc("/verify-human", logWrapper(sessionLoader(verifyHumanHandler)))
	http.HandleFunc("/quiz", logWrapper(sessionLoader(quizHandler)))
	http.HandleFunc("/quiz/answer", logWrapper(sessionLoader(answerHandler)))
	http.HandleFunc("/healthcheck", logWrapper(healthcheckHandler))

	// Static assets
	fs := http.FileServer(http.Dir("./public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	var err error
	db, err = sql.Open("mysql", dbCfg.ConnString(false))
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	initCounts()
	fmt.Printf("Total words: %d\n", totalWords)
	fmt.Printf("Total clues: %d\n", totalClues)

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	fmt.Printf("Random generator is initialized\n")

	session.Init()
	fmt.Println("Sessions initialized")

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", context.ClearHandler(http.DefaultServeMux))
}
