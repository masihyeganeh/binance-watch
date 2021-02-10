package main

import (
	"flag"
	"fmt"
	"github.com/masihyeganeh/binance-watch/internal/binance"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

var symbol = flag.String("symbol", "", "symbol to watch")
var price = flag.Float64("price", 0, "price limit")

func main() {
	flag.Parse()
	log.SetFlags(0)

	if symbol == nil || *symbol == "" || price == nil || *price == 0 {
		fmt.Println("Usage: binance-watch -symbol symbol -price price")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Handling graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	symbols := map[string]bool{strings.ToLower(*symbol): true}

	app, err := binance.New(symbols)
	if err != nil {
		log.Fatal(err)
	}

	defer app.Close()

	done := make(chan struct{})

	go func() {
		err = app.Start(done, interrupt)
		if err != nil {
			log.Println(err.Error())
		}
	}()

	log.Println("Started receiving trades")

	recv := app.Receive()
	for {
		select {
		case <-done:
			return
		case res := <-recv:
			tradePrice, err := strconv.ParseFloat(res.Data.Price, 64)
			if err != nil {
				log.Println("Could not parse price of trade " + res.Data.Price)
			} else if tradePrice > *price {
				log.Printf("received a trade for %s at %f\n", res.Data.Symbol, tradePrice)
			}
		}
	}
}
