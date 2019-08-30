package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/polycache/structures"
	"github.com/vmihailenco/msgpack"
)

func GetTrades(symbols []string) (trades map[string]structures.Trade, err error) {
	if symbols == nil || len(symbols) == 0 {
		return nil, nil
	}

	err = get(fmt.Sprintf(
		"%v/trades?symbols=%v",
		env.GetVar("POLYCACHE_HOST"),
		strings.Join(symbols, ","),
	), &trades)
	return
}

func GetQuotes(symbols []string) (quotes map[string]structures.Quote, err error) {
	if symbols == nil || len(symbols) == 0 {
		return nil, nil
	}

	err = get(fmt.Sprintf(
		"%v/quotes?symbols=%v",
		env.GetVar("POLYCACHE_HOST"),
		strings.Join(symbols, ","),
	), &quotes)
	return
}

func WriteTrade(symbol string, t structures.Trade) error {
	return post(fmt.Sprintf(
		"%v/trades/%v",
		env.GetVar("POLYCACHE_HOST"),
		symbol,
	), t)
}

func WriteQuote(symbol string, q structures.Quote) error {
	return post(fmt.Sprintf(
		"%v/quotes/%v",
		env.GetVar("POLYCACHE_HOST"),
		symbol,
	), q)
}

func GetSnapshot() (snapshot map[string]interface{}, err error) {
	err = get(fmt.Sprintf(
		"%v/snapshot",
		env.GetVar("POLYCACHE_HOST"),
	), &snapshot)
	return
}

func PostSnapshot(snapshot map[string]interface{}, force bool) error {
	return post(fmt.Sprintf(
		"%v/snapshot?force=%v",
		env.GetVar("POLYCACHE_HOST"),
		force,
	), snapshot)
}

func get(url string, output interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	return msgpack.NewDecoder(resp.Body).Decode(&output)
}

func post(url string, body interface{}) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		buf, _ = ioutil.ReadAll(resp.Body)
		return fmt.Errorf("status code: %v (%v)", resp.StatusCode, string(buf))
	}

	return nil
}
