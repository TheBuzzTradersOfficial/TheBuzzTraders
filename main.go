package main

import (
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
	Query   string
	Results *stocks.StockQuote
}

type IndexInfo struct {
	Ticker  []stocks.StockTicker
	Article []*stocks.Article
}

func Symbol() string {
	p := &stocks.StockTicker{}
	return p.Symbol
}

func indexHandler(stockapi *stocks.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		TickerInfo1 := stocks.GetStockTickerInfo("VZ")
		TickerInfo2 := stocks.GetStockTickerInfo("TSLA")
		TickerInfo3 := stocks.GetStockTickerInfo("AMZN")
		TickerInfo4 := stocks.GetStockTickerInfo("AAPL")
		TickerInfo := []stocks.StockTicker{*TickerInfo1, *TickerInfo2, *TickerInfo3, *TickerInfo4}

		var newsList []*stocks.Article

		for i := 0; i < 10; i++ {
			news, err := stockapi.GetArticle(i)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			newsList = append(newsList, news)
		}

		index := IndexInfo{
			Ticker:  TickerInfo,
			Article: newsList,
		}

		buf := &bytes.Buffer{}
		err := templ.ExecuteTemplate(w, "index", index)
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

func searchHandler(stockapi *stocks.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		params := u.Query()
		searchQuery := params.Get("q")

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
		log.Fatal("Env: apiKey must be set")
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := stocks.NewClient(stockClient, apiKey)

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/", indexHandler(stockapi))
	mux.HandleFunc("/News", newsHandler)
	mux.HandleFunc("/search", searchHandler(stockapi))
	http.ListenAndServe(":"+port, mux)
}
