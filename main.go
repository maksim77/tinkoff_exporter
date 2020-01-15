package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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
