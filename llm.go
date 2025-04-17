package main

import (
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
	var apiURLConstruct = fmt.Sprintf("%s/%s?system=%s&model=%s&json=false", apiURL, url.QueryEscape(text), url.QueryEscape(prompt), model)
	var req, err = http.NewRequest("GET", apiURLConstruct, nil)
	if err != nil {
		return "", fmt.Errorf("не удалось сделать запрос: %w", err)
	}

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

	return strings.TrimSpace(strings.Trim(string(str), "`")), nil
}
