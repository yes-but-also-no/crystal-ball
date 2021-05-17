package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/oliveagle/jsonpath"
	"github.com/orakurudata/crystal-ball/cmd/crystal-ball/monitoring"
	"github.com/orakurudata/crystal-ball/configuration"
	"github.com/orakurudata/crystal-ball/contracts"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	MainnetID = big.NewInt(56)
	TestnetID = big.NewInt(97)
)

type Node struct {
	Requests *configuration.Requests
	Web3     *configuration.Web3

	ChainID     *big.Int
	CoreAddress common.Address
	Client      *ethclient.Client
	Core        *contracts.IOrakuruCore
	Registry    *contracts.IAddressRegistry
	Staking     *contracts.IStaking

	FulfillmentMutex *sync.Mutex
}

const (
	AggrTypeMostFrequent = iota
	AggrTypeMedian
	AggrTypeAverage
)

func (n *Node) Start() error {
	address := crypto.PubkeyToAddress(n.Web3.PrivateKey.PublicKey)
	log.Info().Str("wallet", address.String()).Msg("crystal-ball is starting")
	c, err := ethclient.Dial(n.Web3.URL)
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
	n.ChainID = chainID
	n.Client = c
	n.CoreAddress = common.HexToAddress(n.Web3.OrakuruCore)
	n.Core, err = contracts.NewIOrakuruCore(n.CoreAddress, n.Client)
	n.FulfillmentMutex = &sync.Mutex{}
	if err != nil {
		return err
	}
	registryAddr, err := n.Core.AddressRegistry(nil)
	if err != nil {
		return err
	}
	n.Registry, err = contracts.NewIAddressRegistry(registryAddr, c)
	if err != nil {
		return err
	}
	stakingAddr, err := n.Registry.GetStakingAddr(nil)
	if err != nil {
		return err
	}
	n.Staking, err = contracts.NewIStaking(stakingAddr, c)
	if err != nil {
		return err
	}
	oracle, err := n.Staking.IsRegisteredOracle(nil, crypto.PubkeyToAddress(n.Web3.PrivateKey.PublicKey))
	if err != nil {
		return err
	}
	if !oracle {
		log.Error().Caller().Msg("current wallet is not a registered oracle")
	}
	n.Run()
	return nil
}

func (n *Node) Run() {
	go n.RunRequestExecutor()
	go n.updateMonitoringBalance()
}

func (n *Node) executeRequest(url, query string) (string, error) {
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	if query[0] == '$' {
		r.Header.Set("Accept", "application/json")
	} else if query[0] == '/' {
		r.Header.Set("Accept", "application/xml")
	}
	c := &http.Client{}
	c.Timeout = n.Requests.Timeout
	resp, err := c.Do(r)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request execution failed, http status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()

	if query[0] == '$' {
		q, err := jsonpath.Compile(query)
		if err != nil {
			return "", err
		}
		var data interface{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return "", err
		}
		var resp interface{}
		resp, err = q.Lookup(data)
		if err != nil {
			return "", err
		}
		switch r := resp.(type) {
		case string:
			return r, nil
		case float64:
			return strconv.FormatFloat(r, 'f', -1, 64), nil
		default:
			return "", errors.New("invalid jsonpath provided")
		}
	} else if query[0] == '/' {
		doc, err := xmlquery.Parse(bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		nodes, err := xmlquery.QueryAll(doc, query)
		if err != nil {
			return "", err
		}
		if len(nodes) != 1 {
			return "", errors.New("invalid xpath provided")
		}
		return nodes[0].Data, nil
	}
	return "", errors.New("unknown query provided")
}

func sleepUntil(t time.Time) {
	time.Sleep(time.Until(t))
}

func (n *Node) updateMonitoringBalance() {
	ticker := time.Tick(5 * time.Minute)
	for range ticker {
		balance, err := n.Client.BalanceAt(context.Background(), crypto.PubkeyToAddress(n.Web3.PrivateKey.PublicKey), nil)
		if err != nil {
			log.Error().Err(err).Msg("could not update balance")
			continue
		}
		dec := decimal.NewFromBigInt(balance, -18)
		bal, _ := dec.Float64()
		monitoring.AccountBalanceGauge.Set(bal)
	}
}

func validateNumber(number *string) bool {
	_, err := strconv.ParseFloat(*number, 64)
	if err == nil {
		return true
	}
	*number = strings.Replace(*number, ",", ".", 1)
	_, err = strconv.ParseFloat(*number, 64)
	return err == nil
}

func (n *Node) execute(event *contracts.IOrakuruCoreRequested, executionTime time.Time) {
	monitoring.QueueGauge.Inc()
	defer func() {
		monitoring.QueueGauge.Dec()
		monitoring.ExecutedJobsCounter.Inc()
	}()
	sleepUntil(executionTime)
	log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Msg("executing request")
	allowed, err := n.Requests.Filter.ValidateURL(event.DataSource)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("url validation failed, possibly an invalid request - ignoring")
		monitoring.FailedJobsCounter.Inc()
		return
	}
	if !allowed {
		log.Warn().Msg("request violates security policy - ignoring")
		return
	}
	resp, err := n.executeRequest(event.DataSource, event.Selector)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("request execution failed")
		monitoring.FailedJobsCounter.Inc()
		return
	}
	log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Str("result", resp).Msg("request executed successfully")
	if event.AggrType == AggrTypeAverage || event.AggrType == AggrTypeMedian {
		if !validateNumber(&resp) {
			log.Warn().
				Str("id", hexutil.Encode(event.RequestId[:])).
				Str("result", resp).
				Msg("wanted a number, got a string")
			monitoring.FailedJobsCounter.Inc()
			return
		}
	}
	k, err := bind.NewKeyedTransactorWithChainID(n.Web3.PrivateKey, n.ChainID)
	if err != nil {
		log.Error().Err(err).Caller().Msg("cannot create keyed transactor")
		monitoring.FailedJobsCounter.Inc()
		return
	}
	n.FulfillmentMutex.Lock()
	tx, err := n.Core.SubmitResult(k, event.RequestId, resp)
	n.FulfillmentMutex.Unlock()
	if err != nil {
		log.Error().Err(err).Caller().Msg("cannot submit transaction to the network")
		monitoring.FailedJobsCounter.Inc()
		return
	}
	log.Info().Str("id", hexutil.Encode(event.RequestId[:])).Str("tx", tx.Hash().String()).Msg("request fulfilled")
	//sleepUntil(fulfillmentTime)
	// TODO: call fulfill request
}

func (n *Node) collectEvents(startBlock int64) ([]*contracts.IOrakuruCoreRequested, error) {
	num, err := n.Client.BlockNumber(context.Background())
	if err != nil {
		return nil, err
	}
	var out []*contracts.IOrakuruCoreRequested
	for i := startBlock; uint64(i) <= num; i += 4001 {
		end := uint64(i + 4000)
		iter, err := n.Core.FilterRequested(&bind.FilterOpts{
			Start: uint64(i),
			End:   &end,
		}, nil, nil)
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			out = append(out, iter.Event)
		}
		num, err = n.Client.BlockNumber(context.Background())
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (n *Node) pushEvents(events [][32]byte, out chan<- *contracts.IOrakuruCoreRequested) {
	for _, event := range events {
		req, err := n.Core.GetRequest(nil, event)
		if err != nil {
			log.Error().Err(err).Caller().Msg("cannot retrieve event from contract")
			return
		}
		out <- &contracts.IOrakuruCoreRequested{
			RequestId:          event,
			DataSource:         req.DataSource,
			Selector:           req.Selector,
			ExecutionTimestamp: req.ExecutionTimestamp,
			AggrType:           req.AggrType,
		}
	}
}

func (n *Node) RunRequestExecutor() {
	log.Trace().Msg("reloading past events")

	requests, err := n.Core.GetPendingRequests(nil)
	if err != nil {
		log.Error().Err(err).Caller().Msg("cannot get pending requests")
	}

	sink := make(chan *contracts.IOrakuruCoreRequested, len(requests)+100)
	go n.pushEvents(requests, sink)

	log.Trace().Msg("past events were reloaded")

	// TODO: maybe we should unsubscribe when node exits
	_, err = n.Core.WatchRequested(nil, sink, nil, nil)
	if err != nil {
		log.Fatal().Err(err).Caller().Msg("could not subscribe for new events")
	}
	log.Info().Msg("subscribed for new requests")

	for event := range sink {
		// Copy event to pass it into a goroutine
		event := event

		log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Msg("new request received")
		executionTime := time.Unix(event.ExecutionTimestamp.Int64(), 0)
		// FIXME: this time might change on mainnet
		now := time.Now()
		expire := executionTime.Add(1 * time.Minute)
		if !expire.After(now) {
			log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Time("now", now).Time("expire", expire).Msg("event is outdated")
			// Event is expired, skip it
			continue
		}

		go n.execute(event, executionTime)
	}
}
