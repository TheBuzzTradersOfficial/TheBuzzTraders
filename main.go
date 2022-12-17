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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	quote := getQuote("TSLA")

	stockQuote := stocks.StockQuote{
		CurrentPrice:       quote.CurrentPrice,
		HighPOD:            quote.HighPOD,
		LowPOD:             quote.LowPOD,
		OpenPOD:            quote.OpenPOD,
		PreviousClosePrice: quote.PreviousClosePrice,
	}

	buf := &bytes.Buffer{}
	err := templ.ExecuteTemplate(w, "index", stockQuote)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	buf.WriteTo(w)
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
		err = templ.Execute(buf, search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		buf.WriteTo(w)
	}
}

func getQuote(symbol string) *stocks.StockQuote {
	apiKey := os.Getenv("STOCK_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
	}

	stockClient := &http.Client{Timeout: 10 * time.Second}
	stockapi := stocks.NewClient(stockClient, apiKey)

	quote, err := stockapi.FetchQuote(symbol)
	if err != nil {
		return nil
	}

	return quote
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
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/News", newsHandler)
	mux.HandleFunc("/search", searchHandler(stockapi))
	http.ListenAndServe(":"+port, mux)
}
