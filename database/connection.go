package database

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
)

type Conn struct {
	db *sql.DB
}

func OpenConnection(url string) (*Conn, error) {
	db, err := sql.Open("sqlite3", url)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		db: db,
	}
	err = c.CreateSchema()
	return c, err
}

func (c *Conn) Close() error {
	return c.db.Close()
}

func (c *Conn) CreateSchema() error {
	_, err := c.db.Exec("CREATE TABLE IF NOT EXISTS requests (request_id BLOB UNIQUE, data_source TEXT, selector TEXT, execution_timestamp DATETIME, fulfillment_timestamp DATETIME)")
	if err != nil {
		return err
	}
	_, err = c.db.Exec("CREATE TABLE IF NOT EXISTS kv (key TEXT UNIQUE, value TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func (c *Conn) GetRequests() ([]*Request, error) {
	r, err := c.db.Query("SELECT * FROM requests")
	if err != nil {
		return nil, err
	}
	var out []*Request
	for r.Next() {
		item := &Request{}
		err = r.Scan(&item.RequestID, &item.DataSource, &item.Selector, &item.ExecutionTimestamp, &item.FulfillmentTimestamp)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, r.Err()
}

func (c *Conn) AddRequest(r *Request) error {
	_, err := c.db.Exec("INSERT INTO requests VALUES (?, ?, ?, ?, ?)", r.RequestID, r.DataSource, r.Selector, r.ExecutionTimestamp, r.FulfillmentTimestamp)
	return err
}

func (c *Conn) FulfillRequest(requestID []byte) error {
	_, err := c.db.Exec("DELETE FROM requests WHERE request_id = ?", requestID)
	return err
}

func (c *Conn) GetString(key string) (string, error) {
	r := c.db.QueryRow("SELECT value FROM kv WHERE key = ?", key)
	v := ""
	err := r.Scan(&v)
	return v, err
}

func (c *Conn) GetInt(key string) (int64, error) {
	v, err := c.GetString(key)
	if err != nil {
		return 0, err
	}
	vi, err := strconv.ParseInt(v, 10, 64)
	return vi, err
}

func (c *Conn) SetString(key, value string) error {
	_, err := c.GetString(key)
	if err == nil {
		_, err = c.db.Exec("UPDATE kv SET value = ? WHERE key = ?", value, key)
	} else if errors.Is(err, sql.ErrNoRows) {
		_, err = c.db.Exec("INSERT INTO kv VALUES (?, ?)", key, value)
	}
	return err
}

func (c *Conn) SetInt(key string, value int64) error {
	v := strconv.Itoa(int(value))
	return c.SetString(key, v)
}
