package main

import (
	"log"
	"sync"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

type TinkoffCollector struct{}

func (c TinkoffCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- totalAmount
	ch <- stockPrice
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
	ch <- prometheus.MustNewConstMetric(totalAmount, prometheus.GaugeValue, total)

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
			ch <- prometheus.MustNewConstMetric(stockPrice,
				prometheus.GaugeValue,
				value,
				string(p.InstrumentType), p.Ticker, string(p.ExpectedYield.Currency), "1")
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("stock_count", "Stock count", []string{"type", "ticker"}, nil),
				prometheus.GaugeValue,
				p.Balance,
				string(p.InstrumentType), p.Ticker)
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("stock_expected_yield", "Stock expected yield", []string{"type", "ticker", "currency"}, nil),
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
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("currency", "Currency", []string{"currency"}, nil),
				prometheus.GaugeValue, c.Balance, string(c.Currency))
			ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("currency_blocker", "Blocked currency", []string{"currency"}, nil),
				prometheus.GaugeValue, c.Blocked, string(c.Currency))
			wg.Done()
		}(c, ch)
	}
	wg.Wait()
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("total_payin", "Total PayIn", nil, nil), prometheus.GaugeValue, getPayIn(getHistory()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc("total_payout", "Total PayOut", nil, nil), prometheus.GaugeValue, getPayOut(getHistory()))

	tickers := viper.GetStringSlice("tickers")
	for _, t := range tickers {
		wg.Add(1)
		go func(t string, ch chan<- prometheus.Metric) {
			f := getFigi(t)
			price, err := getLastPrice(f.FIGI)
			if err != nil {
				return
			}
			ch <- prometheus.MustNewConstMetric(stockPrice, prometheus.GaugeValue, price, "Stock", t, string(f.Currency), "0")
			wg.Done()
		}(t, ch)
	}
	wg.Wait()
}
