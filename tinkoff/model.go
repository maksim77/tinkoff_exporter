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
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type TinkoffCollector struct {
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
	return &TinkoffCollector{
		totalAmountDesc:        prometheus.NewDesc("total", "Total amount", nil, nil),
		stockPriceDesc:         prometheus.NewDesc("stock", "Stock price", []string{"type", "ticker", "currency", "in_portfolio"}, nil),
		stockCountDesc:         prometheus.NewDesc("stock_count", "Stock count", []string{"type", "ticker"}, nil),
		stockExpectedYieldDesc: prometheus.NewDesc("stock_expected_yield", "Stock expected yield", []string{"type", "ticker", "currency"}, nil),
		currencyDesc:           prometheus.NewDesc("currency", "Currency", []string{"currency"}, nil),
		currencyBlockedDesc:    prometheus.NewDesc("currency_blocked", "Blocked currency", []string{"currency"}, nil),
		totalPayInDesc:         prometheus.NewDesc("total_payin", "Total PayIn", nil, nil),
		totalPayOutDesc:        prometheus.NewDesc("total_payout", "Total PayOut", nil, nil),
		xirrDesc:               prometheus.NewDesc("xirr", "Internal Rate of Return (IRR) for an irregular series of cash flows", nil, nil),
	}
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
	if d := time.Now().Weekday().String(); d == "Sunday" || d == "Saturday" {
		return
	}

	var wg sync.WaitGroup

	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("Fatal error config file: %s \n", err)
		return
	}

	portfolio, err := getPortfolio()
	if err != nil {
		log.Errorf("Cannot get portfolio: %s", err)
		return
	}

	total, err := getTotal(portfolio)
	if err != nil {
		log.Errorf("Get total error: %s", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalAmountDesc, prometheus.GaugeValue, total)

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
				value = lastPrice * 10
			default:
				value = lastPrice
			}

			ch <- prometheus.MustNewConstMetric(c.stockPriceDesc,
				prometheus.GaugeValue,
				value,
				string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency), "1")
			ch <- prometheus.MustNewConstMetric(c.stockCountDesc,
				prometheus.GaugeValue,
				p.Balance,
				string(p.InstrumentType), p.Ticker)
			ch <- prometheus.MustNewConstMetric(c.stockExpectedYieldDesc,
				prometheus.GaugeValue,
				p.ExpectedYield.Value,
				string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency)) //TODO Обработать разные валюты
			wg.Done()
		}(p, ch)
	}

	for _, currency := range portfolio.Currencies {
		wg.Add(1)

		go func(cb sdk.CurrencyBalance, ch chan<- prometheus.Metric) {
			ch <- prometheus.MustNewConstMetric(c.currencyDesc,
				prometheus.GaugeValue, cb.Balance, string(cb.Currency))
			ch <- prometheus.MustNewConstMetric(c.currencyBlockedDesc,
				prometheus.GaugeValue, cb.Blocked, string(cb.Currency))
			wg.Done()
		}(currency, ch)
	}

	hist, err := getHistory()

	if err != nil {
		log.Errorf("Get history error: %s", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalPayInDesc, prometheus.GaugeValue, getPayIn(hist))
	ch <- prometheus.MustNewConstMetric(c.totalPayOutDesc, prometheus.GaugeValue, getPayOut(hist))

	xirr := getXirr(hist, total)

	ch <- prometheus.MustNewConstMetric(c.xirrDesc, prometheus.GaugeValue, xirr)

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

			ch <- prometheus.MustNewConstMetric(c.stockPriceDesc, prometheus.GaugeValue, price, "Stock", t, string(f.Currency), "0")

			wg.Done()
		}(t, ch)
	}

	wg.Wait()
}