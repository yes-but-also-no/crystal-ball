package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/oliveagle/jsonpath"
	"github.com/orakurudata/crystal-ball/cmd/crystal-ball/monitoring"
	"github.com/orakurudata/crystal-ball/configuration"
	"github.com/orakurudata/crystal-ball/contracts"
	"github.com/orakurudata/crystal-ball/secrets"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"io"
	"math/big"
	"net/http"
	"regexp"
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

	ActiveRequests      map[[32]byte]bool
	ActiveRequestsMutex *sync.Mutex
}

const (
	AggrTypeMostFrequent = iota
	AggrTypeMedian
	AggrTypeAverage
)

var (
	JSONPathHotfix = regexp.MustCompile("(?U)\\[\"(.+)\"]")
	SecretRegexp   = regexp.MustCompile(`(?U)\$\$(?P<key>.+):(?P<data>.+)\$\$`)
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
	n.ActiveRequestsMutex = &sync.Mutex{}
	n.ActiveRequests = make(map[[32]byte]bool)
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

func (n *Node) UnwrapSecrets(url string) (string, error) {
	out := SecretRegexp.FindAllStringSubmatch(url, -1)
	for _, match := range out {
		source := match[0]
		key, err := base64.StdEncoding.DecodeString(match[SecretRegexp.SubexpIndex("key")])
		if err != nil {
			return url, err
		}
		ciphertext, err := base64.StdEncoding.DecodeString(match[SecretRegexp.SubexpIndex("data")])
		if err != nil {
			return url, err
		}
		publicKey := secrets.PublicKey(key)
		seed := secrets.Seed(n.Requests.SecretKey)
		value, err := secrets.Decrypt(seed, publicKey, ciphertext)
		if err != nil {
			return url, err
		}
		url = strings.Replace(url, source, value, 1)
	}
	return url, nil
}

func (n *Node) executeRequest(url, query string) (string, error) {
	url, err := n.UnwrapSecrets(url)
	if err != nil {
		log.Warn().Caller().Err(err).Msg("failed to unwrap secrets in URL")
	}
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
		time.Sleep(200 * time.Millisecond)
		resp, err = c.Do(r)
		if err != nil {
			return "", err
		}
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
		query = JSONPathHotfix.ReplaceAllString(query, ".$1")
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

// Gets the last block time
func getLastBlockTime(client *ethclient.Client) (uint64, error) {
	// Get latest block header
	header, err := client.HeaderByNumber(context.Background(), nil)

	// Return zero if failed
	if err != nil {
		return 0, err
	}

	// Return block time
	return header.Time, nil
}

func (n *Node) execute(event *contracts.IOrakuruCoreRequested, executionTime time.Time) {
	monitoring.QueueGauge.Inc()
	defer func() {
		monitoring.QueueGauge.Dec()
		monitoring.ExecutedJobsCounter.Inc()

		n.ActiveRequestsMutex.Lock()
		delete(n.ActiveRequests, event.RequestId)
		n.ActiveRequestsMutex.Unlock()
	}()

	// Perform validation immediately upon receiving request
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

	// Sleep until execution time
	sleepUntil(executionTime)

	// Perform execution like normal
	log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Msg("executing request")

	resp, err := n.executeRequest(event.DataSource, event.Selector)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("request execution failed")
		monitoring.FailedJobsCounter.Inc()
		return
	}
	log.Trace().Str("id", hexutil.Encode(event.RequestId[:])).Str("result", resp).Msg("request executed successfully. waiting to submit.")
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

	// wait for current last block to reach execution timestamp or greater
	for {
		// Get last block time
		lastBlockTime, err := getLastBlockTime(n.Client)
		// If its an error, just sleep 3 seconds like before
		if err != nil {
			log.Warn().Err(err).Msg("could not get latest block time")
			time.Sleep(3 * time.Second)
			break
		}

		// If last block's time implies valid submission on next block, submit
		if lastBlockTime >= uint64(executionTime.Unix()) {
			break
		}

		// Otherwise, sleep a little
		time.Sleep(100 * time.Millisecond)
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
		log.Warn().Msg("waiting 5 seconds before trying to submit the result again")
		time.Sleep(5 * time.Second)
		n.FulfillmentMutex.Lock()
		tx, err = n.Core.SubmitResult(k, event.RequestId, resp)
		n.FulfillmentMutex.Unlock()
		if err != nil {
			monitoring.FailedJobsCounter.Inc()
			return
		}
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
	log.Trace().Msg("past events were reloaded")
}

func (n *Node) RunRequestExecutor() {
	backoff := 5 * time.Second
	for {
		log.Trace().Msg("reloading past events")
		requests, err := n.Core.GetPendingRequests(nil)
		if err != nil {
			log.Error().Err(err).Caller().Msg("cannot get pending requests")
			log.Warn().Dur("backoff", backoff).Msg("waiting before trying again")
			time.Sleep(backoff)
			backoff += 5 * time.Second
			continue
		}

		sink := make(chan *contracts.IOrakuruCoreRequested, len(requests)+100)
		go n.pushEvents(requests, sink)

		// TODO: maybe we should unsubscribe when node exits
		sub, err := n.Core.WatchRequested(nil, sink, nil, nil)
		if err != nil {
			log.Error().Err(err).Caller().Msg("could not subscribe for new events")
			log.Warn().Dur("backoff", backoff).Msg("waiting before trying again")
			time.Sleep(backoff)
			backoff += 5 * time.Second
			continue
		}
		backoff = 5 * time.Second
		log.Info().Msg("subscribed for new requests")
		n.HandlerLoop(sub, sink)
	}
}

func (n *Node) HandlerLoop(sub event.Subscription, sink chan *contracts.IOrakuruCoreRequested) {
	// As far as we I can tell, sometimes nodes drop long-term subscriptions without any notification.
	// We'll resubscribe every 10 hours.
	ticker := time.Tick(10 * time.Hour)
	for {
		select {
		case ev := <-sink:
			n.ActiveRequestsMutex.Lock()
			if _, ok := n.ActiveRequests[ev.RequestId]; ok {
				continue
			}
			n.ActiveRequestsMutex.Unlock()

			evt := ev
			log.Trace().Str("id", hexutil.Encode(evt.RequestId[:])).Msg("new request received")
			executionTime := time.Unix(evt.ExecutionTimestamp.Int64(), 0)
			// FIXME: this time might change on mainnet
			now := time.Now()
			expire := executionTime.Add(1 * time.Minute)
			if !expire.After(now) {
				log.Trace().Str("id", hexutil.Encode(evt.RequestId[:])).Time("now", now).Time("expire", expire).Msg("event is outdated")
				// Event is expired, skip it
				continue
			}

			n.ActiveRequestsMutex.Lock()
			n.ActiveRequests[ev.RequestId] = true
			n.ActiveRequestsMutex.Unlock()
			go n.execute(evt, executionTime)
		case err := <-sub.Err():
			log.Error().Err(err).Caller().Msg("failed receiving events")
			sub.Unsubscribe()
			return
		case <-ticker:
			log.Info().Msg("performing re-subscription to keep connection durable")
			sub.Unsubscribe()
			return
		}
	}
}
