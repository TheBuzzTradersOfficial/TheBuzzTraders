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
	"sort"
	"strings"
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

type StockTickerInit struct {
	Symbol          string
	CurrentPrice    float64
	PercentChange   float64
	Change          float64
	PopularityCount int
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

type StockCandles struct {
	ClosePrices    []float64 `json:"c"`
	HighPrices     []float64 `json:"h"`
	LowPrices      []float64 `json:"l"`
	OpenPrices     []float64 `json:"o"`
	ResponseStatus string    `json:"s"`
	Timestamps     []int64   `json:"t"`
	Volume         []int64   `json:"v"`
}

type ArticleList struct {
	ArticleListItem []Article
}

type GainersLosers []struct {
	Symbol            string  `json:"symbol"`
	Name              string  `json:"name"`
	Change            float64 `json:"change"`
	Price             float64 `json:"price"`
	ChangesPercentage float64 `json:"changesPercentage"`
}

// Creates a client object that allows us to connect to the API
func NewClient(httpClient *http.Client, key string) *Client {
	return &Client{httpClient, key}
}

func (c *Client) fetchAPI(endpoint string) ([]byte, error) {
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

	return body, nil
}

// Connects to the finnhub API and makes a call to the Quote endpoint - returns response from the call and error
func (c *Client) FetchQuote(query string) (*StockQuote, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", url.QueryEscape(query), c.key)
	body, err := c.fetchAPI(endpoint)
	if err != nil {
		return nil, err
	}

	res := &StockQuote{}
	return res, json.Unmarshal(body, res)
}

// Calls the FetchQuote function and returns a StockQuote struct
func GetQuote(symbol string) *StockQuote {
	apiKey := os.Getenv("STOCK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("STOCK_API_KEY2")
		if apiKey == "" {
			log.Fatal("Env: apiKey must be set")
		}
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
	body, err := c.fetchAPI(endpoint)
	if err != nil {
		return nil, err
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

// GetStockTickerInfoNoLimit: Used to get the stock ticker info (no limit because it will call API each time)
func GetStockTickerInfoNoLimit(symbol string) *StockTickerInit {
	quote := GetQuote(symbol)
	tickerInfo := &StockTickerInit{}

	tickerInfo.CurrentPrice = quote.CurrentPrice
	tickerInfo.Symbol = symbol
	tickerInfo.PercentChange = math.Round(quote.PercentChange*100) / 100
	tickerInfo.Change = quote.Change

	return tickerInfo
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
	body, err := c.fetchAPI(endpoint)
	if err != nil {
		return nil, err
	}

	var m []Article
	if err := json.Unmarshal(body, &m); err != nil {
		panic(err)
	}

	return m, nil
}

func (c *Client) GetArticle(articleNum int) (*Article, error) {
	articles, err := c.FetchMarketNews("general")
	if err != nil {
		log.Fatal(err)
	}

	article := articles[articleNum]

	// Check if the title starts with a colon and remove it
	if strings.HasPrefix(article.Headline, ":") {
		article.Headline = strings.TrimPrefix(article.Headline, ":")
	}

	return &article, err
}

func (c *Client) FetchStockCandles(symbol string, resolution int32, from int, to int) (*StockCandles, error) {
	endpoint := fmt.Sprintf("https://finnhub.io/api/v1/stock/candle?symbol=%s&resolution=%d&from=%d&to=%d&token=%s", symbol, resolution, from, to, c.key)
	body, err := c.fetchAPI(endpoint)
	if err != nil {
		return nil, err
	}

	res := &StockCandles{}
	return res, json.Unmarshal(body, res)
}

// TODO: Add GetStockCandles

// Retrieves the stock Gainers and Losers from financialmodelingprep.com api
func (c *Client) FetchGainersLosers(gainLose string) (*GainersLosers, error) {
	endpoint := fmt.Sprintf("https://financialmodelingprep.com/api/v3/stock_market/%s?apikey=%s", gainLose, c.key)
	body, err := c.fetchAPI(endpoint)
	if err != nil {
		return nil, err
	}

	res := &GainersLosers{}
	return res, json.Unmarshal(body, res)
}

// Calls the FetchQuote function and returns a Gainers/Losers struct
// Pass in "gainers" or "losers" as a parameter
func GetGainersLosers(gainLose string) *GainersLosers {
	apiKey := os.Getenv("FMP_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := NewClient(stockClient, apiKey)

	gainersLosers, err := stockapi.FetchGainersLosers(gainLose)
	if err != nil {
		log.Fatal(err)
	}

	// Sorts the stocks to either be the top 10 gainers or top 10 losers
	if gainLose == "gainers" {
		sort.Slice(*gainersLosers, func(i, j int) bool {
			return (*gainersLosers)[i].ChangesPercentage > (*gainersLosers)[j].ChangesPercentage
		})
	} else if gainLose == "losers" {
		sort.Slice(*gainersLosers, func(i, j int) bool {
			return (*gainersLosers)[i].ChangesPercentage < (*gainersLosers)[j].ChangesPercentage
		})
	}

	if len(*gainersLosers) > 10 {
		*gainersLosers = (*gainersLosers)[:10]
	}

	fmt.Println(gainersLosers)
	return gainersLosers
}
