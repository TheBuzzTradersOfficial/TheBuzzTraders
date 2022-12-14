package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Finnhub-Stock-API/finnhub-go"
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

type StockQuote struct {
	CurrentPrice       float64 `json:"c"`
	HighPOD            float64 `json:"h"`
	LowPOD             float64 `json:"l"`
	OpenPOD            float64 `json:"o"`
	PreviousClosePrice float64 `json:"pc"`
	Tag                int     `json:"t"`
}

func getQuote(symbol string) {
	cfg := finnhub.NewConfiguration()
	cfg.AddDefaultHeader("X-Finnhub-Token", os.Getenv("STOCK_API_KEY"))
	finnhubClient := finnhub.NewAPIClient(cfg).DefaultApi

	quote, _, err := finnhubClient.Quote(context.Background(), symbol)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v\n", quote)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templ.ExecuteTemplate(w, "index", &Page{Title: "Home"})
}

func newsHandler(w http.ResponseWriter, r *http.Request) {
	templ.ExecuteTemplate(w, "news", &Page{Title: "News"})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	getQuote("AAPL")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/News", newsHandler)
	http.ListenAndServe(":"+port, mux)
}
