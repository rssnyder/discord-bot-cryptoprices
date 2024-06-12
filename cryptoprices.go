package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	CrytoPricesURL = "https://cryptoprices.cc/%s/%s"
)

func GetCryptoPrices(symbol, module string) (result float64, err error) {

	req, err := http.NewRequest("GET", fmt.Sprintf(CrytoPricesURL, symbol, module), nil)
	if err != nil {
		return
	}

	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("github", "rssnyder/discord-bot-cryptoprices")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	result, err = strconv.ParseFloat(strings.TrimSpace(string(data[:])), 64)
	if err != nil {
		return
	}

	return
}
