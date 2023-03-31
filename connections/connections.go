package connections

import (
	"TheBuzzTraders/stocks"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/gopsql/psql"
	"github.com/jmoiron/sqlx"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	host   = "localhost"
	port   = 5432
	user   = "postgres"
	dbname = "BuzzTradersDB"
)

func ConnectToDB() {
	password := os.Getenv("DB_PASSWORD")

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to BuzzTradersDB")
}

func InsertStockTicker(symbol string) {
	db, err := sqlx.Connect("postgres", "user=postgres dbname=BuzzTradersDB sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	tickerInfo := stocks.GetStockTickerInfo(symbol)

	query := `INSERT INTO "StockTickerIndex"("Symbol", "Change", "Percent_Change", "Current_Price") VALUES ($1, $2, $3, $4) 
			ON CONFLICT ("Symbol") DO UPDATE SET "Symbol" = $1, "Change" = $2, "Percent_Change" = $3, "Current_Price" = $4`
	_, err = db.Exec(query, tickerInfo.Symbol, tickerInfo.Change, tickerInfo.PercentChange, tickerInfo.CurrentPrice)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("\nRow inserted successfully for ticker: " + symbol)
	}
}
