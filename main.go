package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

// AuthRequestParam はアクセストークン取得APIのリクエストパラメータの構造体
type AuthRequestParam struct {
	GrantType    string `json:"grantType"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// AuthResponse はアクセストークン取得APIのレスポンスの構造体
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expires_in"`
	Scope       string `json:"scope"`
	IssuedAt    string `json:"issued_at"`
}

// NlpResponseParam は構文解析APIのリクエストパラメータの構造体
type NlpResponseParam struct {
	Sentence string `json:"sentence"`
	Type     string `json:"type"`
}

// NlpResponse は構文解析APIのレスポンスの構造体
type NlpResponse struct {
	Result []struct {
		Tokens []struct {
			Kana string `json:"kana"`
			Pos  string `json:"pos"`
		} `json:"tokens"`
	} `json:"result"`
	Status  string `json:"staus"`
	Message string `json:"message"`
}

func main() {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					parsedMessage := getParsedWords(message.Text)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(parsedMessage)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

func getParsedWords(words string) string {
	param, err := json.Marshal(NlpResponseParam{
		Sentence: words,
		Type:     "default",
	})

	baseURL := "https://api.ce-cotoha.com/api/dev/nlp/v1/parse"
	req, err := http.NewRequest(
		"POST",
		baseURL,
		bytes.NewBuffer(param),
	)
	if err != nil {
		log.Fatal(err)
	}

	clientID := os.Getenv("COTOHA_CLIENT_ID")
	clientSecret := os.Getenv("COTOHA_CLIENT_SERCRET")

	token := getAccessToken(clientID, clientSecret)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("charset", "UTF-8")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	jsonBytes := ([]byte)(byteArray)
	data := new(NlpResponse)
	if err := json.Unmarshal(jsonBytes, data); err != nil {
		log.Fatal(err)
	}

	var parsedWords []string
	for _, results := range data.Result {
		for _, token := range results.Tokens {
			if token.Pos != "格助詞" && token.Pos != "連用助詞" && token.Pos != "引用助詞" && token.Pos != "終助詞" && token.Pos != "判定詞" {
				parsedWords = append(parsedWords, token.Kana)
			}
		}
	}
	return strings.Join(parsedWords, "..")
}
func getAccessToken(clientID string, clientSecret string) string {
	url := "https://api.ce-cotoha.com/v1/oauth/accesstokens"

	param, err := json.Marshal(AuthRequestParam{
		GrantType:    "client_credentials",
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(param),
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("charset", "UTF-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		log.Fatal(err)
	}
	return authResponse.AccessToken
}
