package session

import (
	"github.com/gorilla/sessions"
	"net/http"
)

type Session struct {
	raw sessions.Session
}

var sessionStore *sessions.CookieStore

func init() {
	sessionStore = sessions.NewCookieStore([]byte("some-secret-asadfewf124r134e"))
}

func Start(w http.ResponseWriter, r *http.Request) (Session, error) {
	var s Session
	rawSession, err := sessionStore.Get(r, "crosscraft")
	if err != nil {
		return s, err
	}

	s.raw = *rawSession

	if s.raw.Values["verified"] == nil {
		s.raw.Values["verified"] = false
	}

	if s.raw.Values["score"] == nil {
		s.raw.Values["score"] = 0
	}

	if s.raw.Values["max"] == nil {
		s.raw.Values["max"] = 0
	}

	if s.raw.Values["exp"] == nil {
		s.raw.Values["exp"] = 0
	}

	if s.raw.Values["lvl"] == nil {
		s.raw.Values["lvl"] = 1
	}

	sessionStore.Save(r, w, &s.raw)

	return s, nil
}

func (s Session) IsVerified() bool {
	if s.raw.Values["verified"] != nil && s.raw.Values["verified"] != false {
		return true
	} else {
		return false
	}
}

func (s Session) SetVerified() {
	s.raw.Values["verified"] = true
}

func (s Session) Save(w http.ResponseWriter, r *http.Request) {
	sessionStore.Save(r, w, &s.raw)
}

func (s Session) GetScore() (int, int, int, int) {
	return s.raw.Values["score"].(int), s.raw.Values["max"].(int), s.raw.Values["exp"].(int), s.raw.Values["lvl"].(int)
}

func (s Session) GetNextLevelCap() int {
	var curlvl = s.raw.Values["lvl"].(int)
	return curlvl * curlvl * 10
}

func (s Session) IncreaseScore() {
	score := s.raw.Values["score"].(int)
	max := s.raw.Values["max"].(int)

	score = score + 1
	if max < score {
		max = score
	}

	s.raw.Values["score"] = score
	s.raw.Values["max"] = max

	lvl := s.raw.Values["lvl"].(int)

	exp := s.raw.Values["exp"].(int)
	exp = exp + score

	if exp >= s.GetNextLevelCap() {
		s.raw.Values["lvl"] = lvl + 1
	}

	s.raw.Values["exp"] = exp
}

func (s Session) ResetScore() {
	s.raw.Values["score"] = 0
}

func SessionLoader(h func(http.ResponseWriter, *http.Request, Session)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := Start(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h(w, r, s)
	}
}