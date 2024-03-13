package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
	http "github.com/bogdanfinn/fhttp"
	"github.com/ddkwork/golibrary/mylog"
)

type Response struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func NewRequest(input string, params structs.Params, prevMessages string) (r *http.Response, err error) {
	client, err := client.NewClient()
	if !mylog.Error(err) {
		return
	}

	model := "mistral"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	temperature := "0.5"
	if params.Temperature != "" {
		temperature = params.Temperature
	}

	topP := "0.5"
	if params.Top_p != "" {
		topP = params.Top_p
	}

	safeInput, err := json.Marshal(input)
	if !mylog.Error(err) {
		return
	}

	data := strings.NewReader(fmt.Sprintf(`{
		"frequency_penalty": 0,
		"messages": [
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"model": "%v",
		"presence_penalty": 0,
		"stream": true,
		"temperature": %v,
		"top_p": %v
	}
	`, prevMessages, string(safeInput), model, temperature, topP))

	req, err := http.NewRequest("POST", "http://localhost:11434/v1/chat/completions", data)
	if !mylog.Error(err) {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+params.ApiKey)

	return client.Do(req)
}

func HandleToken(line string) (token string) {
	//https://521github.com/ollama/ollama/blob/main/openai/op#262
	//_, err = w.ResponseWriter.Write([]byte(fmt.Sprintf("data: %s\n\n", d)))
	//_, err = w.ResponseWriter.Write([]byte("data: [DONE]\n\n"))
	if strings.Contains(line, "data: [DONE]") {
		println()
		mylog.Success("done", "finished")
		return
	}

	obj := "{}"
	if len(line) > 1 {
		split := strings.Split(line, "data: ")
		if len(split) > 1 {
			obj = split[1]
		} else {
			obj = split[0]
		}
	}

	var d Response
	if !mylog.Error(json.Unmarshal([]byte(obj), &d)) {
		return
	}

	if d.Choices != nil {
		token = d.Choices[0].Delta.Content
		return token
	}
	return
}
