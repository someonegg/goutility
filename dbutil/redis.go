// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbutil

import (
	. "github.com/garyburd/redigo/redis"
	"github.com/someonegg/goutility/chanutil"
	"golang.org/x/net/context"
	"time"
)

// RedisPool is a contexted redis pool.
type RedisPool struct {
	p      *Pool
	concur chanutil.Semaphore
}

func NewRedisPool(
	newFn func() (Conn, error),
	testFn func(c Conn, t time.Time) error,
	idleTimeout time.Duration,
	maxConcurrent int) *RedisPool {

	mi := maxConcurrent / 5
	if mi <= 0 {
		mi = 2
	}

	rp := &Pool{
		Dial:         newFn,
		TestOnBorrow: testFn,
		MaxIdle:      mi,
		MaxActive:    0,
		Wait:         false,
		IdleTimeout:  idleTimeout,
	}

	p := &RedisPool{}
	p.p = rp
	if maxConcurrent > 0 {
		p.concur = chanutil.NewSemaphore(maxConcurrent)
	}
	return p
}

func (p *RedisPool) acquireConn(ctx context.Context) error {
	if p.concur == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	// Acquire
	case p.concur <- struct{}{}:
		return nil
	}
}

func (p *RedisPool) releaseConn() {
	if p.concur == nil {
		return
	}

	<-p.concur
}

type proxyConn struct {
	Conn
	p *RedisPool
}

func (c *proxyConn) Close() error {
	c.p.releaseConn()
	return c.Conn.Close()
}

func (p *RedisPool) Get(ctx context.Context) (Conn, error) {
	success := false

	err := p.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			p.releaseConn()
		}
	}()

	c := p.p.Get()
	if err = c.Err(); err != nil {
		c.Close()
		return nil, err
	}

	success = true
	return &proxyConn{Conn: c, p: p}, nil
}

func (p *RedisPool) Close() error {
	return p.p.Close()
}
