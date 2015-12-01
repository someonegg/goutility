// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idgener

import (
	"github.com/garyburd/redigo/redis"
	"github.com/someonegg/goutility/dbutil"
	"golang.org/x/net/context"
	"time"
)

type redisGener struct {
	p *dbutil.RedisPool
	k string
}

// If password isnot empty, then do AUTH.
func NewRedisGener(server, password, idkey string,
	maxConcurrent int) (IDGener, error) {

	dial := func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", server)
		if err != nil {
			return nil, err
		}
		if password != "" {
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
		}
		return c, nil
	}

	testOnBorrow := func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	}

	p := dbutil.NewRedisPool(
		dial,
		testOnBorrow,
		60*time.Second,
		maxConcurrent,
	)
	return &redisGener{p: p, k: idkey}, nil
}

func (g *redisGener) Close() error {
	return g.p.Close()
}

func (g *redisGener) GenID(ctx context.Context) (int64, error) {
	c, err := g.p.Get(ctx)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	id, err := redis.Int64(c.Do("INCR", g.k))
	return id, err
}
