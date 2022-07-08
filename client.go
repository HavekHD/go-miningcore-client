package miningcore

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientOpts are options for the client.
type ClientOpts func(*Client)

// WithoutTLSVerify disables TLS verification.
func WithoutTLSVerfiy() ClientOpts {
	return func(c *Client) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		c.http.Transport = tr
	}
}

// WithTimout sets the default request timeout
func WithTimeout(t time.Duration) ClientOpts {
	return func(c *Client) {
		c.timeout = t
	}
}

// WithJSONEncoder sets the JSON encoder for the client.
func WithJSONEncoder(encoder func(v interface{}) ([]byte, error)) ClientOpts {
	return func(c *Client) {
		c.jsonEncoder = encoder
	}
}

// WithJSONDecoder sets the JSON decoder for the client.
func WithJSONDecoder(decoder func(data []byte, v interface{}) error) ClientOpts {
	return func(c *Client) {
		c.jsonDecoder = decoder
	}
}

// Client represents a client for the miningcore API.
type Client struct {
	timeout     time.Duration
	url         string
	http        *http.Client
	jsonEncoder func(v interface{}) ([]byte, error)
	jsonDecoder func(data []byte, v interface{}) error
}

// New creates a new client for the miningcore API.
func New(url string, opts ...ClientOpts) *Client {
	c := &Client{
		timeout:     time.Second * 20,
		url:         strings.TrimSuffix(url, "/"),
		jsonEncoder: json.Marshal,
		jsonDecoder: json.Unmarshal,
		http:        &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.http = &http.Client{Timeout: c.timeout}
	return c
}

// doRequest performs the actual request to the miningcore API.
func (c *Client) doRequest(ctx context.Context, endpoint, method string, expRes, reqData any, params ...map[string]string) (int, error) {
	callURL, err := buildRequestUrl(c.url, endpoint, params...)
	if err != nil {
		return 0, err
	}

	var dataReq []byte
	if reqData != nil {
		dataReq, err = c.jsonEncoder(reqData)
		if err != nil {
			return 0, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, callURL, bytes.NewBuffer(dataReq))
	if err != nil {
		return 0, err
	}
	if dataReq != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	switch resp.StatusCode {
	case 200:
		if expRes != nil {
			err = c.jsonDecoder(body, expRes)
			if err != nil {
				return 0, err
			}
		}
		return resp.StatusCode, nil

	default:
		return resp.StatusCode, fmt.Errorf("%s", body)
	}
}

func buildRequestUrl(base, endpoint string, params ...map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = endpoint
	if len(params) == 0 {
		return u.String(), nil
	}
	p := url.Values{}
	for k, v := range params[0] {
		p.Set(k, v)
	}
	u.RawQuery = p.Encode()
	return u.String(), nil
}
