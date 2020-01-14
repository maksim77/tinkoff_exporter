package main

import (
	"context"
	"log"
	"net/http"
	"time"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("endpoint", "/metrics")
	viper.SetDefault("port", 8000)
	viper.SetDefault("token", "CHANGEME")
	viper.SetDefault("tickers", []string{})
	viper.SetDefault("сurrencies", map[string]string{"usd": "BBG0013HGFT4", "eur": "BBG0013HJJ31"})
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Please write config file")
			err = viper.SafeWriteConfigAs("config.yaml")
			if err != nil {
				log.Fatalf("Error write config sample: %s", err)
			}
		} else {
			log.Printf("Fatal error config file: %s \n", err)
		}
	}

	if viper.GetString("token") == "CHANGEME" {
		log.Fatal("You must specify the correct token!")
	}

	c := TinkoffCollector{}
	prometheus.MustRegister(c)
	http.Handle(viper.GetString("endpoint"), promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+viper.GetString("port"), nil))
}

func getTotal(portfolio sdk.Portfolio) (total float64, err error) {
	totalPositions, err := getTotalPositions(portfolio.Positions)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	totalCurrencies, err := getTotalCurrencies(portfolio.Currencies)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	total = totalPositions + totalCurrencies
	return total, nil
}

func getPortfolio() (sdk.Portfolio, error) {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	portfolio, err := client.Portfolio(ctx)
	if err != nil {
		return sdk.Portfolio{}, err
	}
	return portfolio, nil
}

func getTotalPositions(positions []sdk.PositionBalance) (totalPositions float64, err error) {
	for _, p := range positions {
		lastPrice, err := getLastPrice(p.FIGI)
		if err != nil {
			return 0, nil
		}
		if p.InstrumentType == "Bond" {
			totalPositions = totalPositions + (p.Balance * ((lastPrice * 10) + (p.AveragePositionPrice.Value - p.AveragePositionPriceNoNkd.Value)))
		} else {
			totalPositions = totalPositions + (p.Balance * lastPrice)
		}
	}
	return
}

func getTotalCurrencies(currencies []sdk.CurrencyBalance) (totalCurrencies float64, err error) {
	сurrs := viper.GetStringMapString("сurrencies")

	for _, c := range currencies {
		switch c.Currency {
		case "RUB":
			totalCurrencies = totalCurrencies + c.Balance
		case "USD":
			lastPrice, err1 := getLastPrice(сurrs["usd"])
			if err != nil {
				log.Println(err)
				return 0, err1
			}
			totalCurrencies = totalCurrencies + (c.Balance * lastPrice)
		case "EUR":
			lastPrice, err1 := getLastPrice(сurrs["usd"])
			if err != nil {
				log.Println(err)
				return 0, err1
			}
			totalCurrencies = totalCurrencies + (c.Balance * lastPrice)
		}
	}
	return
}

func getLastPrice(figi string) (float64, error) {
	client := sdk.NewRestClient(viper.GetString("token"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orderbook, err := client.Orderbook(ctx, 1, figi)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return orderbook.LastPrice, nil
}

func getHistory() []sdk.Operation {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	operations, err := client.Operations(ctx, time.Date(2019, time.October, 1, 0, 0, 0, 0, time.UTC), time.Now(), "")
	if err != nil {
		log.Fatal(err)
	}
	var listOperations []sdk.Operation
	for _, o := range operations {
		switch o.OperationType {
		case "PayIn":
			listOperations = append(listOperations, o)
		case "PayOut":
			listOperations = append(listOperations, o)
		}
	}
	return listOperations
}

func getPayIn(ops []sdk.Operation) float64 {
	var total float64
	for _, o := range ops {
		switch o.OperationType {
		case "PayIn":
			total += o.Payment
		}
	}
	return total
}

func getPayOut(ops []sdk.Operation) float64 {
	var total float64
	for _, o := range ops {
		switch o.OperationType {
		case "PayOut":
			total -= o.Payment
		}
	}
	return total
}

func getFigi(ticker string) sdk.Instrument {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instruments, err := client.SearchInstrumentByTicker(ctx, ticker)
	if err != nil {
		log.Fatalf("Unable to get figi by ticker: %s", err)
	}
	if len(instruments) != 1 {
		log.Fatalf("Multiple instriments return by one ticker: %v", instruments)
	}
	return instruments[0]
}
