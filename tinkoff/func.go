/*
Copyright © 2020 Maksim Syomochkin <maksim77ster@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package tinkoff

import (
	"context"
	"math"
	"time"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/maksim77/goxirr"
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
		log.Error(err)
		return 0, err
	}

	total = totalPositions + totalCurrencies

	return total, nil
}

func getPortfolio(accountId string) (sdk.Portfolio, error) {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	portfolio, err := client.Portfolio(ctx, accountId)
	if err != nil {
		return sdk.Portfolio{}, err
	}

	return portfolio, nil
}

func getTotalPositions(positions []sdk.PositionBalance) (totalPositions float64, err error) {
	for _, p := range positions {
		lastPrice, err := getLastPrice(p.FIGI)
		if err != nil {
			return 0, err
		}

		if p.InstrumentType == "Bond" {
			totalPositions += (p.Balance * (lastPrice + (p.AveragePositionPrice.Value - p.AveragePositionPriceNoNkd.Value)))
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
			totalCurrencies += c.Balance
		case "USD":
			lastPrice, err := getLastPrice(сurrs["usd"])

			if err != nil {
				log.Println(err)

				return 0, err
			}

			totalCurrencies += c.Balance * lastPrice
		case "EUR":
			lastPrice, err := getLastPrice(сurrs["eur"])

			if err != nil {
				log.Println(err)
				return 0, err
			}

			totalCurrencies += c.Balance * lastPrice
		}
	}

	return
}

func getLastPrice(figi string) (float64, error) {
	client := sdk.NewRestClient(viper.GetString("token"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	defer cancel()

	orderbook, err := client.Orderbook(ctx, 1, figi)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	return orderbook.LastPrice, nil
}

func getHistory(id string) ([]sdk.Operation, error) {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	operations, err := client.Operations(ctx, id, time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), time.Now(), "")
	if err != nil {
		return []sdk.Operation{}, err
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

	return listOperations, nil
}

func getPayIn(ops []sdk.Operation) float64 {
	var total float64

	for _, o := range ops {
		if o.OperationType == "PayIn" {
			total += o.Payment
		}
	}

	return total
}

func getPayOut(ops []sdk.Operation) float64 {
	var total float64

	for _, o := range ops {
		if o.OperationType == "PayOut" {
			total -= o.Payment
		}
	}

	return total
}

func getFigi(ticker string) (sdk.SearchInstrument, error) {
	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	instruments, err := client.SearchInstrumentByTicker(ctx, ticker)
	if err != nil {
		log.Errorf("Unable to get figi by ticker: %s\n", err)
		return sdk.SearchInstrument{}, err
	}

	if len(instruments) != 1 {
		log.Printf("Multiple instriments return by one ticker: %v\n", instruments)
		return sdk.SearchInstrument{}, err
	}

	return instruments[0], nil
}

func getXirr(operations []sdk.Operation, total float64) float64 {
	var ts goxirr.Transactions

	for _, o := range operations {
		if o.OperationType == "PayIn" || o.OperationType == "PayOut" {
			var t goxirr.Transaction
			t.Date = o.DateTime

			switch o.OperationType {
			case "PayIn":
				t.Cash = 0 - o.Payment
			case "PayOut":
				t.Cash = math.Abs(o.Payment)
			}

			ts = append(ts, t)
		}
	}

	reverseTransactionsList(ts)

	ts = append(ts, goxirr.Transaction{Date: time.Now(), Cash: total})

	return goxirr.Xirr(ts)
}

func reverseTransactionsList(in goxirr.Transactions) {
	for i := 0; i < len(in)/2; i++ {
		j := len(in) - i - 1
		in[i], in[j] = in[j], in[i]
	}
}
