package main

import (
	"log"
	"sync"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

var (
	totalAmountDesc        = prometheus.NewDesc("total", "Total amount", nil, nil)
	stockPriceDesc         = prometheus.NewDesc("stock", "Stock price", []string{"type", "ticker", "currency", "in_portfolio"}, nil)
	stockCountDesc         = prometheus.NewDesc("stock_count", "Stock count", []string{"type", "ticker"}, nil)
	stockExpectedYieldDesc = prometheus.NewDesc("stock_expected_yield", "Stock expected yield", []string{"type", "ticker", "currency"}, nil)
	currencyDesc           = prometheus.NewDesc("currency", "Currency", []string{"currency"}, nil)
	currencyBlockedDesc    = prometheus.NewDesc("currency_blocked", "Blocked currency", []string{"currency"}, nil)
	totalPayInDesc         = prometheus.NewDesc("total_payin", "Total PayIn", nil, nil)
	totalPayOutDesc        = prometheus.NewDesc("total_payout", "Total PayOut", nil, nil)
)

type TinkoffCollector struct{}

func (c TinkoffCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- totalAmountDesc
	ch <- stockPriceDesc
	ch <- stockCountDesc
	ch <- stockExpectedYieldDesc
	ch <- currencyDesc
	ch <- currencyBlockedDesc
	ch <- totalPayInDesc
	ch <- totalPayOutDesc
}

func (c TinkoffCollector) Collect(ch chan<- prometheus.Metric) {
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Fatal error config file: %s \n", err)
		return
	}
	var wg sync.WaitGroup
	portfolio, err := getPortfolio()
	if err != nil {
		log.Printf("Cannot get portfolio: %s", err)
		return
	}
	total, err := getTotal(portfolio)
	if err != nil {
		log.Printf("Get total error: %s", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(totalAmountDesc, prometheus.GaugeValue, total)

	for _, p := range portfolio.Positions {
		wg.Add(1)
		go func(p sdk.PositionBalance, ch chan<- prometheus.Metric) {
			var value float64
			lastPrice, err := getLastPrice(p.FIGI)
			if err != nil {
				return
			}
			switch p.InstrumentType {
			case "Bond":
				value = lastPrice * 10
			default:
				value = lastPrice
			}
			ch <- prometheus.MustNewConstMetric(stockPriceDesc,
				prometheus.GaugeValue,
				value,
				string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency), "1")
			ch <- prometheus.MustNewConstMetric(stockCountDesc,
				prometheus.GaugeValue,
				p.Balance,
				string(p.InstrumentType), p.Ticker)
			ch <- prometheus.MustNewConstMetric(stockExpectedYieldDesc,
				prometheus.GaugeValue,
				p.ExpectedYield.Value,
				string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency)) //TODO Обработать разные валюты
			wg.Done()
		}(p, ch)
	}
	wg.Wait()

	for _, c := range portfolio.Currencies {
		wg.Add(1)
		go func(c sdk.CurrencyBalance, ch chan<- prometheus.Metric) {
			ch <- prometheus.MustNewConstMetric(currencyDesc,
				prometheus.GaugeValue, c.Balance, string(c.Currency))
			ch <- prometheus.MustNewConstMetric(currencyBlockedDesc,
				prometheus.GaugeValue, c.Blocked, string(c.Currency))
			wg.Done()
		}(c, ch)
	}
	wg.Wait()
	ch <- prometheus.MustNewConstMetric(totalPayInDesc, prometheus.GaugeValue, getPayIn(getHistory()))
	ch <- prometheus.MustNewConstMetric(totalPayOutDesc, prometheus.GaugeValue, getPayOut(getHistory()))

	tickers := viper.GetStringSlice("tickers")
	for _, t := range tickers {
		wg.Add(1)
		go func(t string, ch chan<- prometheus.Metric) {
			f, err := getFigi(t)
			if err != nil {
				return
			}
			price, err := getLastPrice(f.FIGI)
			if err != nil {
				return
			}
			ch <- prometheus.MustNewConstMetric(stockPriceDesc, prometheus.GaugeValue, price, "Stock", t, string(f.Currency), "0")
			wg.Done()
		}(t, ch)
	}
	wg.Wait()
}
