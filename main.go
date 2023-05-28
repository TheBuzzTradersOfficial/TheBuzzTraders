package main

import (
	"TheBuzzTraders/connections"
	"TheBuzzTraders/stocks"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

var templ = func() *template.Template {
	t := template.New("")
	err := filepath.Walk("./templates/", func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".html") {
			fmt.Println(path)
			_, err = t.ParseFiles(path)
			if err != nil {
				fmt.Println(err)
			}
		}
		return err
	})

	if err != nil {
		panic(err)
	}
	return t
}()

type Page struct {
	Title string
}

type StockSearch struct {
	Query        string
	Results      *stocks.StockQuote
	ErrorMessage string
}

type IndexInfo struct {
	Ticker  []stocks.StockTicker
	Article []*stocks.Article
}

func Symbol() string {
	p := &stocks.StockTicker{}
	return p.Symbol
}

func GetNewsArticles(stockapi *stocks.Client, n int) ([]*stocks.Article, error) {
	var newsList []*stocks.Article
	for i := 0; i < n; i++ {
		news, err := stockapi.GetArticle(i)
		if err != nil {
			return nil, err
		}
		newsList = append(newsList, news)
	}
	return newsList, nil
}

// TODO: Add the top 4 most searched stocks in the past hour to the main page stock tickers
func indexHandler(stockapi *stocks.Client) http.HandlerFunc {
	connections.GetPopularStocks()

	return func(w http.ResponseWriter, r *http.Request) {
		symbols := []string{"QQQ", "TSLA", "AMZN", "AAPL"}
		var TickerInfo []stocks.StockTicker
		for _, symbol := range symbols {
			tickerInfo := stocks.GetStockTickerInfo(symbol)
			TickerInfo = append(TickerInfo, *tickerInfo)
		}

		newsList, err := GetNewsArticles(stockapi, 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		index := IndexInfo{
			Ticker:  TickerInfo,
			Article: newsList,
		}

		buf := &bytes.Buffer{}
		err = templ.ExecuteTemplate(w, "index", index)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		buf.WriteTo(w)
	}
}

func newsHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := templ.ExecuteTemplate(w, "news", &Page{Title: "News"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	buf.WriteTo(w)
}

// Used to verify that the value searched is a valid stock symbol
// TODO: If page is refreshed do not increase popularity count for stock
func checkSearchQuery(stockapi *stocks.Client, query string) bool {
	listOfSymbols := stockapi.GetStockSymbols()
	found := false

	for _, s := range listOfSymbols {
		if s == query {
			found = true
			break
		}
	}

	if found {
		fmt.Println("Search query was a valid symbol")
		return true
	} else {
		fmt.Println("Search query was invalid")
		return false
	}
}

// Used for the search bar to search for different Stock Symbols
func searchHandler(stockapi *stocks.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Conenct to DB to update search count Popularity_Count
		db, err := sqlx.Connect("postgres", "user=postgres dbname=BuzzTradersDB sslmode=disable")
		if err != nil {
			log.Fatalln(err)
		}
		defer db.Close()

		params := u.Query()
		searchQuery := strings.ToUpper(params.Get("q"))

		if checkSearchQuery(stockapi, searchQuery) == true {
			connections.InsertStockTicker(searchQuery)

			// Gets the value of the Popularity Count
			var popularityCount string
			err = db.QueryRow(`SELECT "Popularity_Count" FROM "StockTickerIndex" WHERE "Symbol" = $1`, searchQuery).Scan(&popularityCount)
			if err != nil {
				log.Fatal(err)
			}

			// Increase the value of Popularity_Count by 1
			changePopCount, err := db.Exec(`UPDATE "StockTickerIndex" SET "Popularity_Count" = "Popularity_Count" + 1 WHERE "Symbol" = $1`, searchQuery)
			if err != nil {
				log.Fatal(err)
			}

			rowsAffected, err := changePopCount.RowsAffected()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%d row(s) updated for symbol: %s (Popularity_Count)", rowsAffected, searchQuery)
		} else if checkSearchQuery(stockapi, searchQuery) == false {
			referer := r.Header.Get("Referer")
			if referer != "" {
				query := url.Values{}
				query.Set("error", "Invalid search query")
				separator := "?"
				if strings.Contains(referer, "?") {
					separator = "&"
				}
				http.Redirect(w, r, referer+separator+query.Encode(), http.StatusSeeOther)
			} else {
				// TODO: Show splash page error
				log.Fatal(err)
			}
		}

		results, err := stockapi.FetchQuote(searchQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		search := &StockSearch{
			Query:   searchQuery,
			Results: results,
		}

		buf := &bytes.Buffer{}
		err = templ.ExecuteTemplate(w, "search", search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		buf.WriteTo(w)
	}
}

func refreshStockTickerInfo() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		connections.UpdateTable(`"StockTickerIndex"`)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiKey := os.Getenv("STOCK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("STOCK_API_KEY2")
		if apiKey == "" {
			log.Fatal("Env: apiKey must be set")
		}
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := stocks.NewClient(stockClient, apiKey)

	fs := http.FileServer(http.Dir("assets"))

	connections.ConnectToDB()
	// TODO: Run this every 6 minutes and update Gainers/Losers on index.html
	stocks.GetGainersLosers("gainers")

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/", indexHandler(stockapi))
	mux.HandleFunc("/News", newsHandler)
	mux.HandleFunc("/search", searchHandler(stockapi))

	// go refreshStockTickerInfo()

	http.ListenAndServe(":"+port, mux)
}
