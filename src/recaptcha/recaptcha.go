package recaptcha

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type verificationResponse struct {
	Success     bool     `json:"success"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

var Key string
var Secret string

func Verify(token string) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {Secret},
		"response": {token},
	})
	defer response.Body.Close()
	if err != nil {
		return false, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	result := new(verificationResponse)
	err = json.Unmarshal(body, result)
	if err != nil {
		return false, err
	}

	return result.Success, nil
}
