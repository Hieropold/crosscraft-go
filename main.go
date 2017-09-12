package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"html/template"
	"net/http"
	"os"
	"recaptcha"
	"session"
	"strconv"
	"time"
	"word"
)

type QuizPage struct {
	Word     word.Word
	Score    int
	MaxScore int
	Exp      int
	Lvl      int
	NextCap  int
	Progress int
}

type SuccessPage struct {
	Score int
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

func checkAccess(s session.Session) bool {
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
	score, max, exp, lvl := s.GetScore()

	// Load random word
	var randomWord word.Word
	randomWord, err = word.GetRandomWord()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = randomWord.ApplyIncorrectClues(4)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	next := s.GetNextLevelCap()
	prev := s.GetPreviousLevelCap()
	progress := (float64(exp-prev) / float64(next-prev)) * 100

	var quizPage QuizPage = QuizPage{
		Word:     randomWord,
		Score:    score,
		MaxScore: max,
		Exp:      exp,
		Lvl:      lvl,
		NextCap:  next,
		Progress: int(progress),
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
	isCorrect, err = word.IsCorrectClue(wid, cid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isCorrect {
		s.IncreaseScore()
		s.Save(w, r)
		score, _, _, _ := s.GetScore()
		var successPage SuccessPage = SuccessPage{
			Score: score,
		}
		err = successTpl.ExecuteTemplate(w, "content", successPage)
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

	http.HandleFunc("/", logWrapper(session.SessionLoader(indexHandler)))
	http.HandleFunc("/verify-human", logWrapper(session.SessionLoader(verifyHumanHandler)))
	http.HandleFunc("/quiz", logWrapper(session.SessionLoader(quizHandler)))
	http.HandleFunc("/quiz/answer", logWrapper(session.SessionLoader(answerHandler)))
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

	err = word.Bootstrap(db)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Total words: %d\n", word.TotalWords())
	fmt.Printf("Total clues: %d\n", word.TotalClues())

	fmt.Println("Crosscraft server is listening on port 8080")

	http.ListenAndServe(":8080", context.ClearHandler(http.DefaultServeMux))
}
