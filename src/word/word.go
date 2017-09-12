package word

import (
	"database/sql"
	"math/rand"
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

var totalWords int
var totalClues int

var db *sql.DB

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func Bootstrap(dbh *sql.DB) error {
	db = dbh
	return initCounts()
}

func GetRandomWord() (Word, error) {
	var word Word
	var err error

	offset := getRandomOffset(totalWords)

	word, err = getWordByOffset(offset)
	if err != nil {
		return word, err
	}

	return word, nil
}

func (w *Word) ApplyIncorrectClues(count int) error {
	clues := make([]Clue, 1)

	// Select random correct clue
	clues[0] = w.Clues[getRandomOffset(len(w.Clues))]

	// Load random clues
	var randomClues []Clue
	randomClues, err := getRandomIncorrectClues(w.Wid, count)
	if err != nil {
		return err
	}
	clues = append(clues, randomClues...)

	// Shuffle clues
	shuffled := make([]Clue, len(clues))
	perm := rand.Perm(len(clues))
	for i, v := range perm {
		shuffled[v] = clues[i]
	}

	w.Clues = shuffled

	return nil
}

func IsCorrectClue(wid int, cid int) (bool, error) {
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

func TotalWords() int {
	return totalWords
}

func TotalClues() int {
	return totalClues
}

func getRandomOffset(limit int) int {
	return random.Intn(limit)
}

func initCounts() error {
	err := db.QueryRow("SELECT COUNT(*) AS total FROM words").Scan(&totalWords)
	if err != nil {
		return err
	}

	err = db.QueryRow("SELECT COUNT(*) AS total FROM clues").Scan(&totalClues)
	if err != nil {
		return err
	}

	return nil
}
