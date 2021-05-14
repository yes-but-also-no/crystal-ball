package database

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/sha3"
	"testing"
	"time"
)

func TestOpenConnection(t *testing.T) {
	_, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
}

func TestConn_SetString(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	err = c.SetString("test", "123")
	if err != nil {
		t.Fatalf("SetString returned an error: %v", err)
	}
	err = c.SetString("test", "345")
	if err != nil {
		t.Fatalf("SetString returned an error: %v", err)
	}
}

func TestConn_GetString(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	_, err = c.GetString("test")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetString failed, want = sql.ErrNoRows, got = %v", err)
	}
	err = c.SetString("test", "123")
	if err != nil {
		t.Fatalf("SetString returned an error: %v", err)
	}
	v, err := c.GetString("test")
	if err != nil {
		t.Fatalf("GetString returned an error: %v", err)
	}
	if v != "123" {
		t.Fatalf("GetString returned wrong value, want = 123, got = %v", v)
	}
	err = c.SetString("test", "456")
	if err != nil {
		t.Fatalf("SetString returned an error: %v", err)
	}
	v, err = c.GetString("test")
	if err != nil {
		t.Fatalf("GetString returned an error: %v", err)
	}
	if v != "456" {
		t.Fatalf("GetString returned wrong value, want = 456, got = %v", v)
	}
}

func TestConn_SetInt(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	err = c.SetInt("test", 123)
	if err != nil {
		t.Fatalf("SetInt returned an error: %v", err)
	}
	err = c.SetInt("test", 456)
	if err != nil {
		t.Fatalf("SetInt returned an error: %v", err)
	}
}

func TestConn_GetInt(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	_, err = c.GetInt("test")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetInt failed, want = sql.ErrNoRows, got = %v", err)
	}
	err = c.SetInt("test", 123)
	if err != nil {
		t.Fatalf("SetInt returned an error: %v", err)
	}
	v, err := c.GetInt("test")
	if err != nil {
		t.Fatalf("GetInt returned an error: %v", err)
	}
	if v != 123 {
		t.Fatalf("GetInt returned wrong value, want = 123, got = %v", v)
	}
	err = c.SetInt("test", 456)
	if err != nil {
		t.Fatalf("SetInt returned an error: %v", err)
	}
	v, err = c.GetInt("test")
	if err != nil {
		t.Fatalf("GetInt returned an error: %v", err)
	}
	if v != 456 {
		t.Fatalf("GetInt returned wrong value, want = 456, got = %v", v)
	}
	err = c.SetString("test", "notint")
	if err != nil {
		t.Fatalf("SetString returned an error: %v", err)
	}
	_, err = c.GetInt("test")
	if err == nil {
		t.Fatalf("GetInt returned success, expected error")
	}
}

func TestConn_AddRequest(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	hash := sha3.Sum256([]byte("hello"))
	err = c.AddRequest(&Request{
		RequestID:            hash[:],
		DataSource:           "Source",
		Selector:             "Selector",
		ExecutionTimestamp:   time.Now().Add(30 * time.Minute),
		FulfillmentTimestamp: time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("AddRequest returned an error: %v", err)
	}
}

func TestConn_GetRequests(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	hash := sha3.Sum256([]byte("hello"))
	req := &Request{
		RequestID:            hash[:],
		DataSource:           "Source",
		Selector:             "Selector",
		ExecutionTimestamp:   time.Now().Add(30 * time.Minute),
		FulfillmentTimestamp: time.Now().Add(1 * time.Hour),
	}
	err = c.AddRequest(req)
	if err != nil {
		t.Fatalf("AddRequest returned an error: %v", err)
	}
	reqs, err := c.GetRequests()
	if err != nil {
		t.Fatalf("GetRequests returned an error: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %v", len(reqs))
	}
	// TODO: properly compare received and stored values
	// DeepEqual doesn't work in this case
}

func TestConn_FulfillRequest(t *testing.T) {
	c, err := OpenConnection("file::memory:")
	if err != nil {
		t.Fatalf("OpenConnection returned an error: %v", err)
	}
	hash := sha3.Sum256([]byte("hello"))
	req := &Request{
		RequestID:            hash[:],
		DataSource:           "Source",
		Selector:             "Selector",
		ExecutionTimestamp:   time.Now().Add(30 * time.Minute),
		FulfillmentTimestamp: time.Now().Add(1 * time.Hour),
	}
	err = c.AddRequest(req)
	if err != nil {
		t.Fatalf("AddRequest returned an error: %v", err)
	}
	hash[0] = 0
	err = c.AddRequest(req)
	if err != nil {
		t.Fatalf("AddRequest returned an error: %v", err)
	}
	reqs, err := c.GetRequests()
	if err != nil {
		t.Fatalf("GetRequests returned an error: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 request, got %v", len(reqs))
	}
	err = c.FulfillRequest(hash[:])
	if err != nil {
		t.Fatalf("FulfillRequest returned an error: %v", err)
	}
	reqs, err = c.GetRequests()
	if err != nil {
		t.Fatalf("GetRequests returned an error: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 requests, got %v", len(reqs))
	}
}
