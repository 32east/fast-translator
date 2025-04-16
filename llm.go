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

func Generate(text string) (string, error) {
	var apiURLConstruct = fmt.Sprintf("%s/%s?system=%s&model=%s&json=true", apiURL, url.QueryEscape(text), url.QueryEscape(prompt), model)
	var req, err = http.NewRequest("GET", apiURLConstruct, nil)
	if err != nil {
		return "", fmt.Errorf("не удалось сделать запрос: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Keep-Alive")

	var minTime = math.Max(float64(utf8.RuneCountInString(text)/23), 5)
	var client = &http.Client{
		Timeout: time.Duration(minTime * 1000 * 1000 * 1000),
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("не удалось подключиться к серверу: %w", err)
	}
	defer resp.Body.Close()

	var str, strErr = io.ReadAll(resp.Body)
	if strErr != nil {
		return "", strErr
	}

	var convertedStr = strings.TrimSpace(strings.Trim(string(str), "`"))
	var finalStr string
	var decoded = make(map[string]any)
	json.Unmarshal([]byte(convertedStr), &decoded)

	switch {
	case decoded["translation"] != nil:
		finalStr = decoded["translation"].(string)
	case decoded["response"] != nil:
		finalStr = decoded["response"].(string)
	default:
		finalStr = convertedStr
	}

	finalStr, _ = url.QueryUnescape(finalStr)

	return finalStr, nil
}
