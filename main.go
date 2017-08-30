package main

import (
	"fmt"
	"net/http"
	"html/template"
	"database/sql"
	"math/rand"
	_ "github.com/go-sql-driver/mysql"
	sessions "github.com/gorilla/sessions"
	"time"
	"strconv"
	"github.com/gorilla/context"
	"recaptcha"
	"os"
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

type DBConfig struct {
	User string
	Password string
	DBName string
	Host string
}

func (c DBConfig) ConnString() string {
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

var sessionStore = sessions.NewCookieStore([]byte("some-secret-asadfewf124r134e"))

func isVerified(r *http.Request) (bool, error) {
	session, err := sessionStore.Get(r, "crosscraft")
	if err != nil {
		return false, err
	}

	if session.Values["verified"] != nil && session.Values["verified"] != false {
		return true, nil
	}

	return false, nil
}

func checkAccess(w http.ResponseWriter, r *http.Request) (bool, error) {
	isVerified, err := isVerified(r)
	if err != nil {
		return false, err
	}
	if isVerified {
		return true, nil
	}


	return false, nil
}

func logWrapper(h func (http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func (w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h(w, r)
		duration := time.Now().Sub(startTime)
		fmt.Printf("Request %s: %d ns\n", r.URL, duration)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var verified bool

	verified, err = isVerified(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type WelcomeData struct {
		IsVerified bool
		RecaptchaKey string
	}
	var data WelcomeData
	data.IsVerified = verified
	data.RecaptchaKey = recaptcha.Key
	err = welcomeTpl.ExecuteTemplate(w, "content", data);
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func verifyHumanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	r.ParseForm()

	token := r.Form["g-recaptcha-response"][0]

	success, err := recaptcha.Verify(token)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if (success) {
		session, err := sessionStore.Get(r, "crosscraft")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		session.Values["verified"] = true
		sessionStore.Save(r, w, session)
		http.Redirect(w, r, "/quiz", http.StatusTemporaryRedirect)
		return
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
}

func quizHandler(w http.ResponseWriter, r *http.Request) {

	var err error

	granted, _ := checkAccess(w, r)
	if (!granted) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Load random word
	var randomWord Word
	randomWord, err = getRandomWord()
	if (err != nil) {
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

func answerHandler(w http.ResponseWriter, r *http.Request) {

	granted, _ := checkAccess(w, r)
	if (!granted) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	params := r.URL.Query()

	if (params["wid"] == nil || params["cid"] == nil) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var err error
	var wid int
	var cid int

	wid, err = strconv.Atoi(params["wid"][0])
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cid, err = strconv.Atoi(params["cid"][0])
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var isCorrect bool
	isCorrect, err = isCorrectClue(wid, cid)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if (isCorrect) {
		err = successTpl.ExecuteTemplate(w, "content", nil)
	} else {
		err = failTpl.ExecuteTemplate(w, "content", nil)
	}
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check DB connection
	err := db.Ping()
	if (err != nil) {
		fmt.Printf("Healthcheck error: %s", err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("I'm OK"))
}

func getRandomOffset(limit int) (int) {
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

	recaptcha.Key = os.Getenv("CROSSCRAFT_RECAPTCHA_KEY")
	if (recaptcha.Key == "") {
		fmt.Printf("Missing reCaptcha token\n")
		return
	}
	recaptcha.Secret = os.Getenv("CROSSCRAFT_RECAPTCHA_SECRET")
	if (recaptcha.Secret == "") {
		fmt.Printf("Missing reCaptcha secret\n")
		return
	}
	fmt.Printf("reCaptcha token: %s\n", recaptcha.Key)
	fmt.Printf("reCaptcha secret: %s\n", recaptcha.Secret)

	var dbCfg DBConfig
	dbCfg.User = os.Getenv("CROSSCRAFT_DB_USER")
	dbCfg.Password = os.Getenv("CROSSCRAFT_DB_PASSWORD")
	dbCfg.DBName = os.Getenv("CROSSCRAFT_DB_NAME")

	http.HandleFunc("/", logWrapper(indexHandler))
	http.HandleFunc("/verify-human", logWrapper(verifyHumanHandler))
	http.HandleFunc("/quiz", logWrapper(quizHandler))
	http.HandleFunc("/quiz/answer", logWrapper(answerHandler))
	http.HandleFunc("/healthcheck", logWrapper(healthcheckHandler))

	// Static assets
	fs := http.FileServer(http.Dir("./public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	var err error
	db, err = sql.Open("mysql", dbCfg.ConnString())
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

	http.ListenAndServe(":8080", context.ClearHandler(http.DefaultServeMux))
}
