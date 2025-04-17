package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultModel = "openai"
const defaultLanguage = "Русский"
const apiURL = "https://text.pollinations.ai"

type AvailableModel struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Provider         string   `json:"provider"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
	Vision           bool     `json:"vision,omitempty"`
	Audio            bool     `json:"audio,omitempty"`
	Uncensored       bool     `json:"uncensored,omitempty"`
}

type Request struct {
	URL        string        `json:"url"`
	Method     string        `json:"method"`
	MaxTimeout time.Duration `json:"timeout"`
	JSON       bool          `json:"json"`
}

func newRequest(request *Request) ([]byte, error) {
	var req, err = http.NewRequest(request.Method, request.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось сделать запрос: %w", err)
	}

	req.Header.Set("Connection", "Keep-Alive")

	if request.JSON {
		req.Header.Set("Content-Type", "application/json")
	}

	var client = &http.Client{
		Timeout: request.MaxTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к серверу: %w", err)
	}
	defer resp.Body.Close()

	var str, strErr = io.ReadAll(resp.Body)
	if strErr != nil {
		return nil, strErr
	}
	return str, nil

}

func GetAvailableModels() ([]AvailableModel, error) {
	var apiURLConstruct = fmt.Sprintf("%s/models", apiURL)
	var str, err = newRequest(&Request{
		URL:        apiURLConstruct,
		Method:     http.MethodGet,
		MaxTimeout: 180 * time.Second,
		JSON:       true,
	})

	if err != nil {
		return nil, err
	}

	var availableModels []AvailableModel
	json.Unmarshal(str, &availableModels)
	return availableModels, nil
}

func Generate(text string) (string, error) {
	var apiURLConstruct = fmt.Sprintf("%s/%s?system=%s&defaultModel=%s&json=false", apiURL, url.PathEscape(text), url.PathEscape(prompt), defaultModel)
	var minTime = math.Max(float64(utf8.RuneCountInString(text)/23), 5)
	var str, err = newRequest(&Request{
		URL:        apiURLConstruct,
		Method:     http.MethodGet,
		MaxTimeout: time.Duration(minTime * 1000 * 1000 * 1000),
	})

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.Trim(string(str), "`")), nil
}
