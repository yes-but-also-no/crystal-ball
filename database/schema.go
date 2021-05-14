package database

import "time"

type Request struct {
	RequestID            []byte
	DataSource           string
	Selector             string
	ExecutionTimestamp   time.Time
	FulfillmentTimestamp time.Time
}

type KV struct {
	Key   string
	Value string
}
