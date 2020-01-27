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
package cmd

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	"tinkoff_exporter/tinkoff"

	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "tinkoff_exporter",
	Short: "Data exporter for OpenAPI Tinkoff Investments",
	Long:  `Data exporter for OpenAPI Tinkoff Investments`,

	Run: func(cmd *cobra.Command, args []string) {
		c := tinkoff.NewTinkoffCollector()
		prometheus.MustRegister(c)
		http.Handle(viper.GetString("endpoint"), promhttp.Handler())
		log.Fatal(http.ListenAndServe(":"+viper.GetString("port"), nil))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	viper.SetDefault("endpoint", "/metrics")
	viper.SetDefault("port", 8000)
	viper.SetDefault("token", "CHANGEME")
	viper.SetDefault("tickers", []string{})
	viper.SetDefault("сurrencies", map[string]string{"usd": "BBG0013HGFT4", "eur": "BBG0013HJJ31"})

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("TINKOFF_EXPORTER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
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
}
