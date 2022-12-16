package stocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client struct {
	http *http.Client
	key  string
}

type StockQuote struct {
	CurrentPrice       float64 `json:"c"`
	HighPOD            float64 `json:"h"`
	LowPOD             float64 `json:"l"`
	OpenPOD            float64 `json:"o"`
	PreviousClosePrice float64 `json:"pc"`
	Tag                int     `json:"t"`
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
