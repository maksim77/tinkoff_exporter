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
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type TinkoffCollector struct {
	sync.Mutex

	accountIDs map[sdk.AccountType]string

	totalAmountDesc        *prometheus.Desc
	stockPriceDesc         *prometheus.Desc
	stockCountDesc         *prometheus.Desc
	stockExpectedYieldDesc *prometheus.Desc
	currencyDesc           *prometheus.Desc
	currencyBlockedDesc    *prometheus.Desc
	totalPayInDesc         *prometheus.Desc
	totalPayOutDesc        *prometheus.Desc
	xirrDesc               *prometheus.Desc
}

func NewTinkoffCollector() *TinkoffCollector {
	tc := &TinkoffCollector{
		accountIDs:             make(map[sdk.AccountType]string),
		totalAmountDesc:        prometheus.NewDesc("total", "Total amount", []string{"account"}, nil),
		stockPriceDesc:         prometheus.NewDesc("stock", "Stock price", []string{"type", "ticker", "currency", "in_portfolio", "account"}, nil),
		stockCountDesc:         prometheus.NewDesc("stock_count", "Stock count", []string{"type", "ticker", "account"}, nil),
		stockExpectedYieldDesc: prometheus.NewDesc("stock_expected_yield", "Stock expected yield", []string{"type", "ticker", "currency", "account"}, nil),
		currencyDesc:           prometheus.NewDesc("currency", "Currency", []string{"currency", "account"}, nil),
		currencyBlockedDesc:    prometheus.NewDesc("currency_blocked", "Blocked currency", []string{"currency", "account"}, nil),
		totalPayInDesc:         prometheus.NewDesc("total_payin", "Total PayIn", []string{"account"}, nil),
		totalPayOutDesc:        prometheus.NewDesc("total_payout", "Total PayOut", []string{"account"}, nil),
		xirrDesc:               prometheus.NewDesc("xirr", "Internal Rate of Return (IRR) for an irregular series of cash flows", []string{"account"}, nil),
	}

	client := sdk.NewRestClient(viper.GetString("token"))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	accounts, err := client.Accounts(ctx)
	if err != nil {
		log.Fatalf("Cannot get accounts: %s", err)
	}

	for _, id := range accounts {
		tc.accountIDs[id.Type] = id.ID
	}

	return tc
}

func (c TinkoffCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalAmountDesc
	ch <- c.stockPriceDesc
	ch <- c.stockCountDesc
	ch <- c.stockExpectedYieldDesc
	ch <- c.currencyDesc
	ch <- c.currencyBlockedDesc
	ch <- c.totalPayInDesc
	ch <- c.totalPayOutDesc
	ch <- c.xirrDesc
}

func (c TinkoffCollector) Collect(ch chan<- prometheus.Metric) {
	c.Lock()
	defer c.Unlock()

	if d := time.Now().Weekday().String(); d == "Sunday" || d == "Saturday" {
		return
	}

	var wg sync.WaitGroup

	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("Fatal error config file: %s \n", err)
		return
	}

	for accountName, accountID := range c.accountIDs {
		portfolio, err := getPortfolio(accountID)
		if err != nil {
			log.Errorf("Cannot get portfolio: %s", err)
			return
		}

		total, err := getTotal(portfolio)

		if err != nil {
			log.Errorf("Get total error: %s", err)
			return
		}

		ch <- prometheus.MustNewConstMetric(c.totalAmountDesc, prometheus.GaugeValue, total, string(accountName))

		for _, p := range portfolio.Positions {
			wg.Add(1)

			go func(p sdk.PositionBalance, ch chan<- prometheus.Metric) {
				var value float64

				lastPrice, err := getLastPrice(p.FIGI)
				if err != nil {
					log.Errorf("Get last price error: %s", err)
					return
				}

				switch p.InstrumentType {
				case "Bond":
					value = lastPrice
				default:
					value = lastPrice
				}

				ch <- prometheus.MustNewConstMetric(c.stockPriceDesc,
					prometheus.GaugeValue,
					value,
					string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency), "1", string(accountName))
				ch <- prometheus.MustNewConstMetric(c.stockCountDesc,
					prometheus.GaugeValue,
					p.Balance,
					string(p.InstrumentType), p.Ticker, string(accountName))
				ch <- prometheus.MustNewConstMetric(c.stockExpectedYieldDesc,
					prometheus.GaugeValue,
					p.ExpectedYield.Value,
					string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency), string(accountName)) //TODO Обработать разные валюты
				wg.Done()
			}(p, ch)
		}

		for _, currency := range portfolio.Currencies {
			wg.Add(1)

			go func(cb sdk.CurrencyBalance, ch chan<- prometheus.Metric) {
				ch <- prometheus.MustNewConstMetric(c.currencyDesc,
					prometheus.GaugeValue, cb.Balance, string(cb.Currency), string(accountName))
				ch <- prometheus.MustNewConstMetric(c.currencyBlockedDesc,
					prometheus.GaugeValue, cb.Blocked, string(cb.Currency), string(accountName))
				wg.Done()
			}(currency, ch)
		}

		hist, err := getHistory(accountID)

		if err != nil {
			log.Errorf("Get history error: %s", err)
			return
		}

		ch <- prometheus.MustNewConstMetric(c.totalPayInDesc, prometheus.GaugeValue, getPayIn(hist), string(accountName))
		ch <- prometheus.MustNewConstMetric(c.totalPayOutDesc, prometheus.GaugeValue, getPayOut(hist), string(accountName))

		xirr := getXirr(hist, total)

		ch <- prometheus.MustNewConstMetric(c.xirrDesc, prometheus.GaugeValue, xirr, string(accountName))

		tickers := viper.GetStringSlice("tickers")
		for _, t := range tickers {
			wg.Add(1)

			go func(t string, ch chan<- prometheus.Metric) {
				f, err := getFigi(t)
				if err != nil {
					log.Errorf("Get FIGI error: %s", err)
					return
				}

				price, err := getLastPrice(f.FIGI)
				if err != nil {
					log.Errorf("Get last price error: %s", err)
					return
				}

				ch <- prometheus.MustNewConstMetric(c.stockPriceDesc, prometheus.GaugeValue, price, "Stock", t, string(f.Currency), "0", string(accountName))

				wg.Done()
			}(t, ch)
		}

		wg.Wait()
	}
}
