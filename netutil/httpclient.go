// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netutil

import (
	"github.com/someonegg/goutility/chanutil"
	"golang.org/x/net/context"
	"io"
	"net"
	. "net/http"
	"net/url"
	"time"
)

// HttpClient is a contexted http client.
type HttpClient struct {
	ts     *Transport
	hc     *Client
	concur chanutil.Semaphore
}

// if maxConcurrent == 0, no limit on concurrency.
func NewHttpClient(maxConcurrent int, timeout time.Duration) *HttpClient {
	mi := maxConcurrent / 5
	if mi <= 0 {
		mi = DefaultMaxIdleConnsPerHost
	}
	ts := &Transport{
		Proxy: ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConnsPerHost: mi,
	}
	hc := &Client{
		Transport: ts,
		Timeout:   timeout,
	}

	c := &HttpClient{}
	c.ts = ts
	c.hc = hc
	if maxConcurrent > 0 {
		c.concur = chanutil.NewSemaphore(maxConcurrent)
	}
	return c
}

func (c *HttpClient) acquireConn(ctx context.Context) error {
	if c.concur == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	// Acquire
	case c.concur <- struct{}{}:
		return nil
	}
}

func (c *HttpClient) releaseConn() {
	if c.concur == nil {
		return
	}

	<-c.concur
}

func (c *HttpClient) Do(ctx context.Context,
	req *Request) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Do(req)
}

func (c *HttpClient) Get(ctx context.Context,
	url string) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Get(url)
}

func (c *HttpClient) Head(ctx context.Context,
	url string) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Head(url)
}

func (c *HttpClient) Post(ctx context.Context,
	url string, bodyType string, body io.Reader) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Post(url, bodyType, body)
}

func (c *HttpClient) PostForm(ctx context.Context,
	url string, data url.Values) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.PostForm(url, data)
}

func (c *HttpClient) Close() error {
	c.ts.CloseIdleConnections()
	return nil
}
