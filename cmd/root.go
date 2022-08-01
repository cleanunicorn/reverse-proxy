/*
Copyright © 2022 Daniel Luca

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"reverse-proxy/proxy"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var log *logrus.Logger = logrus.New()

var rootCmd = &cobra.Command{
	Use:   "reverse-proxy",
	Short: "A brief description of your application",
	Long:  `This is a reverse proxy that will forward requests to a destination.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyHandler, err := proxy.New(viper.GetString("destination"), viper.GetInt("port"))
		if err != nil {
			return err
		}

		// Enable TLS / HTTPS
		if viper.GetBool("tls") {
			proxyHandler.EnableTls()
		}

		// Enable logging
		if viper.GetString("log-level") != "" {
			switch viper.GetString("log-level") {
			case "trace":
				log.SetLevel(logrus.TraceLevel)
			case "debug":
				log.SetLevel(logrus.DebugLevel)
			case "info":
				log.SetLevel(logrus.InfoLevel)
			case "warn":
				log.SetLevel(logrus.WarnLevel)
			case "error":
				log.SetLevel(logrus.ErrorLevel)
			case "fatal":
				log.SetLevel(logrus.FatalLevel)
			case "panic":
				log.SetLevel(logrus.PanicLevel)
			default:
				proxyHandler.SetLogger(log)
			}
		}

		// Start the proxy
		err = proxyHandler.Start()
		if err != nil {
			return err
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.reverse-proxy.yaml)")

	// Listen port
	rootCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))

	// Destination
	rootCmd.Flags().StringP("destination", "d", "http://localhost:8080", "Destination to proxy to")
	viper.BindPFlag("destination", rootCmd.Flags().Lookup("destination"))

	// Use TLS / HTTPS
	rootCmd.Flags().BoolP("tls", "t", false, "Use self signed TLS / HTTPS")
	viper.BindPFlag("tls", rootCmd.Flags().Lookup("tls"))

	// Set log level
	rootCmd.Flags().StringP("log-level", "l", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
	viper.BindPFlag("log-level", rootCmd.Flags().Lookup("log-level"))

	// Set up proxy defaults
	viper.SetDefault("destination", "http://localhost:8080")
	viper.SetDefault("listen-port", 8080)

	// Initialize the logger
	log.SetFormatter(&logrus.TextFormatter{})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".reverse-proxy" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".reverse-proxy")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
