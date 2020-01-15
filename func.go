package main

import (
	"context"
	"time"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

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
			totalPositions += (p.Balance * ((lastPrice * 10) + (p.AveragePositionPrice.Value - p.AveragePositionPriceNoNkd.Value)))
		} else {
			totalPositions += (p.Balance * lastPrice)
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

	operations, err := client.Operations(ctx, time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), time.Now(), "")
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

func getFigi(ticker string) (sdk.Instrument, error) {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instruments, err := client.SearchInstrumentByTicker(ctx, ticker)
	if err != nil {
		log.Printf("Unable to get figi by ticker: %s\n", err)
		return sdk.Instrument{}, err
	}

	if len(instruments) != 1 {
		log.Printf("Multiple instriments return by one ticker: %v\n", instruments)
		return sdk.Instrument{}, err
	}

	return instruments[0], nil
}
