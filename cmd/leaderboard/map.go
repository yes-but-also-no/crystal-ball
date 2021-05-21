package main

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sort"
	"sync"
)

type Validators struct {
	scores        map[common.Address]uint64
	responseTimes map[common.Address]*big.Int
	requests      map[common.Address]*big.Int
	mutex         sync.RWMutex
}

func NewValidators() Validators {
	return Validators{
		scores:        make(map[common.Address]uint64),
		responseTimes: make(map[common.Address]*big.Int),
		requests:      make(map[common.Address]*big.Int),
		mutex:         sync.RWMutex{},
	}
}

func (v *Validators) AddScore(address common.Address, score uint64, delay uint64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.scores[address] += score
	if v.responseTimes[address] == nil {
		v.responseTimes[address] = big.NewInt(0)
		v.requests[address] = big.NewInt(0)
	}
	v.responseTimes[address].Add(v.responseTimes[address], big.NewInt(int64(delay)))
	v.requests[address].Add(v.requests[address], big.NewInt(1))
}

func (v *Validators) Collect() Leaderboard {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	result := make(Leaderboard, 0, len(v.scores))
	for k, val := range v.scores {
		average, _ := big.NewFloat(0).Quo(big.NewFloat(float64(v.responseTimes[k].Uint64())),
			big.NewFloat(float64(v.requests[k].Uint64()))).Float64()
		result = append(result, LeaderboardEntry{
			Address:      k.Hex(),
			Score:        val,
			ResponseTime: average,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})
	return result
}
