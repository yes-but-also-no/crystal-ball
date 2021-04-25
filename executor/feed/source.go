package feed

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/orakurudata/crystal-ball/configuration"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Parser func([]byte, configuration.Parser) (float64, error)

var (
	parsers = map[string]Parser{
		"json": jsonParser,
	}

	ErrNoSuchKey       = errors.New("no such key was found")
	ErrInvalidIndex    = errors.New("invalid index")
	ErrIndexOutOfRange = errors.New("index out of range")
	ErrUnexpectedValue = errors.New("unexpected value")
	ErrInvalidEndValue = errors.New("invalid end value")
)

func ExecuteSource(source configuration.Source, arguments map[string]string) (float64, error) {
	source.URL = configuration.ExpandVariables(source.URL, arguments)
	req, err := http.NewRequest(strings.ToUpper(source.Method), source.URL, bytes.NewReader([]byte{}))
	if err != nil {
		return 0, err
	}
	for k, v := range source.Headers {
		v = configuration.ExpandVariables(v, arguments)
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	source.Parser.Path = configuration.ExpandVariables(source.Parser.Path, arguments)
	return ExecuteParser(respBody, source.Parser)
}

func ExecuteParser(data []byte, parser configuration.Parser) (float64, error) {
	return parsers[parser.Type](data, parser)
}

func jsonParser(data []byte, parser configuration.Parser) (float64, error) {
	v := map[string]interface{}{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return 0, err
	}
	split := strings.Split(parser.Path, "]")
	for i, v := range split {
		split[i] = strings.Replace(v, "[", "", 1)
	}
	var current interface{}
	current = v
	for i, element := range split {
		if i == len(split)-1 {
			// last element in split is guaranteed to be empty
			break
		}
		switch v := current.(type) {
		case map[string]interface{}:
			value, ok := v[element]
			if !ok {
				return 0, ErrNoSuchKey
			}
			current = value
		case []interface{}:
			value, err := strconv.ParseInt(element, 10, 64)
			if err != nil {
				return 0, ErrInvalidIndex
			}
			if int(value) < 0 {
				return 0, ErrIndexOutOfRange
			}
			if int(value) >= len(v) {
				return 0, ErrIndexOutOfRange
			}
			current = v[value]
		default:
			return 0, ErrUnexpectedValue
		}
	}
	result := 0.0
	switch v := current.(type) {
	case int64:
		result = float64(v)
	case float32:
		result = float64(v)
	case float64:
		// should always be float64, but anything might happen!
		result = v
	default:
		return 0, ErrInvalidEndValue
	}
	return result, nil
}
