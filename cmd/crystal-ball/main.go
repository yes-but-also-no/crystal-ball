package main

import (
	"github.com/orakurudata/crystal-ball/cmd/crystal-ball/monitoring"
	"github.com/orakurudata/crystal-ball/configuration"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

func getenv(env, def string) string {
	v := os.Getenv(env)
	if v == "" {
		return def
	}
	return v
}

func loadLogLevel(level string) {
	lv, _ := zerolog.ParseLevel(strings.ToLower(level))
	if lv != zerolog.NoLevel {
		zerolog.SetGlobalLevel(lv)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func main() {
	configDirectory := getenv("CB_CONFIG_DIR", "etc/")
	loadLogLevel(getenv("CB_LOG_LEVEL", "info"))
	prettyLogging := getenv("CB_PRETTY_LOG", "true")
	prometheusHost := getenv("MONITORING_HOST", ":9000")
	if prettyLogging == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	go monitoring.StartMonitoring(prometheusHost)

	requestsFile, err := os.Open(path.Join(configDirectory, "requests.yml"))
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to open requests configuration")
	}
	requestsConfig, err := configuration.ParseRequests(requestsFile)
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to parse requests configuration")
	}
	_ = requestsFile.Close()

	web3File, err := os.Open(path.Join(configDirectory, "web3.yml"))
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to open web3 configuration")
	}
	web3Config, err := configuration.ParseWeb3(web3File)
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to parse web3 configuration")
	}
	_ = web3File.Close()

	node := Node{
		Requests: requestsConfig,
		Web3:     web3Config,
	}
	err = node.Start()
	if err != nil {
		log.Error().Err(err).Caller().Msg("failed to start node")
		return
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info().Msg("received exit signal, stopping")
}
