package stocks

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "github.com/gopsql/psql"
	"github.com/jmoiron/sqlx"
)

type Client struct {
	*http.Client
	key string
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
	tableName     struct{} `pg:"StockTickerIndex"`
	Symbol        string   `pg:"Symbol"`
	CurrentPrice  string   `pg:"Current_Price"`
	PercentChange string   `pg:"Percent_Change"`
	Change        string   `pg:"Change"`
}

type StockSymbol struct {
	Symbol string `json:"symbol"`
}

type Article struct {
	Category string `json:"category"`
	Datetime int    `json:"datetime"`
	Headline string `json:"headline"`
	ID       int    `json:"id"`
	Image    string `json:"image"`
	Related  string `json:"related"`
	Source   string `json:"source"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
}

type ArticleList struct {
	ArticleListItem []Article
}

// Creates a client object that allows us to connect to the API
func NewClient(httpClient *http.Client, key string) *Client {
	return &Client{httpClient, key}
}

// Connects to the finnhub API and makes a call to the Quote endpoint - returns response from the call and error
func (c *Client) FetchQuote(query string) (*StockQuote, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", url.QueryEscape(query), c.key)
	resp, err := c.Get(endpoint)
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

// Calls the FetchQuote function and returns a StockQuote struct
func GetQuote(symbol string) *StockQuote {
	apiKey := os.Getenv("STOCK_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := NewClient(stockClient, apiKey)

	quote, err := stockapi.FetchQuote(symbol)
	if err != nil {
		log.Fatal(err)
	}

	return quote
}

func (c *Client) FetchStockSymbols() ([]StockSymbol, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/stock/symbol?exchange=US&token=%s", c.key)
	resp, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(string(body))
	}

	var m []StockSymbol
	if err := json.Unmarshal(body, &m); err != nil {
		panic(err)
	}

	return m, nil
}

func (c *Client) GetStockSymbols() []string {
	symbols, err := c.FetchStockSymbols()
	if err != nil {
		log.Fatal(err)
	}

	var symbolList []string
	for _, symbol := range symbols {
		symbolList = append(symbolList, string(symbol.Symbol))
	}

	return symbolList[0:]
}

// Calls the GetQuote function in order to return info for the Stock Tickers on the index page
// This function returns a specific struct StockTicker for that part of the application
func GetStockTickerInfo(symbol string) *StockTicker {
	tickerInfo := &StockTicker{}

	db, err := sqlx.Connect("postgres", "user=postgres dbname=BuzzTradersDB sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	var currentPrice string
	err = db.QueryRow(`SELECT "Current_Price" FROM "StockTickerIndex" WHERE "Symbol" = $1`, symbol).Scan(&currentPrice)
	if err != nil {
		log.Fatal(err)
	}

	var percentChange string
	err = db.QueryRow(`SELECT "Percent_Change" FROM "StockTickerIndex" WHERE "Symbol" = $1`, symbol).Scan(&percentChange)
	if err != nil {
		log.Fatal(err)
	}

	var change string
	err = db.QueryRow(`SELECT "Change" FROM "StockTickerIndex" WHERE "Symbol" = $1`, symbol).Scan(&change)
	if err != nil {
		log.Fatal(err)
	}

	tickerInfo.CurrentPrice = currentPrice
	tickerInfo.Symbol = symbol
	tickerInfo.PercentChange = percentChange
	tickerInfo.Change = change

	return tickerInfo
}

// Connects to the finnhub API and makes a call to the Market News endpoint - returns response from the call and error
func (c *Client) FetchMarketNews(query string) ([]Article, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/news?category=%s&token=%s", url.QueryEscape(query), c.key)
	resp, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(string(body))
	}

	var m []Article
	if err := json.Unmarshal(body, &m); err != nil {
		panic(err)
	}

	return m, err
}

func (c *Client) GetArticle(articleNum int) (*Article, error) {
	articles, err := c.FetchMarketNews("general")
	if err != nil {
		log.Fatal(err)
	}

	article := articles[articleNum]

	return &article, err
}
