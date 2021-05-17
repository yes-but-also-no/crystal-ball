package main

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orakurudata/crystal-ball/configuration"
	"github.com/orakurudata/crystal-ball/contracts"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/big"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var (
	MainnetID = big.NewInt(56)
	TestnetID = big.NewInt(97)

	ChainID = big.NewInt(0)

	FulfillMutex = sync.Mutex{}
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
	if prettyLogging == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	web3File, err := os.Open(path.Join(configDirectory, "web3.yml"))
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to open web3 configuration")
	}
	web3Config, err := configuration.ParseWeb3(web3File)
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("failed to parse web3 configuration")
	}
	_ = web3File.Close()

	err = runExecutor(web3Config)
	log.Fatal().Err(err).Caller().Msg("request executor crashed")
}

func pushEvents(core *contracts.IOrakuruCore, events [][32]byte, out chan<- *contracts.IOrakuruCoreRequested) {
	for _, event := range events {
		req, err := core.GetRequest(nil, event)
		if err != nil {
			log.Error().Err(err).Caller().Msg("cannot retrieve event from contract")
			return
		}
		out <- &contracts.IOrakuruCoreRequested{
			RequestId:          event,
			DataSource:         req.DataSource,
			Selector:           req.Selector,
			ExecutionTimestamp: req.ExecutionTimestamp,
		}
	}
}

func sleepUntil(t time.Time) {
	time.Sleep(time.Until(t))
}

func fulfillEvent(core *contracts.IOrakuruCore, event *contracts.IOrakuruCoreRequested, key *ecdsa.PrivateKey) {
	fulfillmentTime := time.Unix(event.ExecutionTimestamp.Int64(), 0).Add(65 * time.Second)
	sleepUntil(fulfillmentTime)
	k, err := bind.NewKeyedTransactorWithChainID(key, ChainID)
	if err != nil {
		log.Error().Err(err).Msg("could not create transactor")
		return
	}
	FulfillMutex.Lock()
	tx, err := core.FulfillRequest(k, event.RequestId)
	FulfillMutex.Unlock()
	if err != nil {
		log.Warn().Err(err).Caller().Msg("could not fulfill request the first time, retying in a random amount of seconds")
		time.Sleep(time.Duration(rand.Intn(5)+2) * time.Second)
		tx, err = core.FulfillRequest(k, event.RequestId)
		if err != nil {
			log.Error().Err(err).Caller().Msg("could not fulfill request the second time as well")
			return
		}
	}
	log.Info().Str("id", hexutil.Encode(event.RequestId[:])).Str("tx", tx.Hash().Hex()).Msg("request is fulfilled")
}

func runExecutor(web3 *configuration.Web3) error {
	address := crypto.PubkeyToAddress(web3.PrivateKey.PublicKey)
	log.Info().Str("wallet", address.String()).Msg("request-fulfiller is starting")
	c, err := ethclient.Dial(web3.URL)
	if err != nil {
		return err
	}
	chainID, err := c.ChainID(context.Background())
	if err != nil {
		return err
	}
	switch {
	case chainID.Cmp(MainnetID) == 0:
		log.Info().Msg("mainnet endpoint detected")
	case chainID.Cmp(TestnetID) == 0:
		log.Info().Msg("testnet endpoint detected")
	default:
		log.Warn().Msg("endpoint network is unknown")
	}
	ChainID = chainID
	coreAddress := common.HexToAddress(web3.OrakuruCore)
	core, err := contracts.NewIOrakuruCore(coreAddress, c)
	if err != nil {
		return err
	}

	pending, err := core.GetPendingRequests(nil)
	if err != nil {
		log.Error().Err(err).Caller().Msg("could not rewind old requests")
	}
	sink := make(chan *contracts.IOrakuruCoreRequested, len(pending)+100)

	_, err = core.WatchRequested(nil, sink, nil, nil)
	if err != nil {
		return err
	}
	go pushEvents(core, pending, sink)

	for event := range sink {
		event := event
		log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Msg("received an event")
		go fulfillEvent(core, event, web3.PrivateKey)
	}
	return nil
}
