package stocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	http *http.Client
	key  string
}

type StockQuote struct {
	CurrentPrice       float64 `json:"c"`
	Change             float64 `json:"d"`
	PercentChange      float64 `json:"dp"`
	HighPOD            float64 `json:"h"`
	LowPOD             float64 `json:"l"`
	OpenPOD            float64 `json:"o"`
	PreviousClosePrice float64 `json:"pc"`
	Tag                int     `json:"t"`
}

type StockTicker struct {
	Symbol        string
	CurrentPrice  float64
	PercentChange float64
	Change        float64
}

func NewClient(httpClient *http.Client, key string) *Client {
	return &Client{httpClient, key}
}

func (c *Client) FetchQuote(query string) (*StockQuote, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", url.QueryEscape(query), c.key)
	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(string(body))
	}

	res := &StockQuote{}
	return res, json.Unmarshal(body, res)
}

func GetQuote(symbol string) *StockQuote {
	apiKey := os.Getenv("STOCK_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := NewClient(stockClient, apiKey)

	quote, err := stockapi.FetchQuote(symbol)
	if err != nil {
		return nil
	}

	return quote
}

func GetStockTickerInfo(symbol string) *StockTicker {
	quote := GetQuote(symbol)
	tickerInfo := &StockTicker{}

	tickerInfo.CurrentPrice = math.Round(quote.CurrentPrice*100) / 100
	tickerInfo.Symbol = symbol
	tickerInfo.PercentChange = math.Round(quote.PercentChange*100) / 100
	tickerInfo.Change = math.Round(quote.Change*100) / 100

	return tickerInfo
}
