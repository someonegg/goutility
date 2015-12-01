// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idgener

import (
	"golang.org/x/net/context"
	"sync/atomic"
)

type memoryGener struct {
	next int64
}

// This gener never fails.
func NewMemoryGener() IDGener {
	return &memoryGener{}
}

func (g *memoryGener) Close() error {
	return nil
}

func (g *memoryGener) GenID(ctx context.Context) (int64, error) {
	id := atomic.AddInt64(&g.next, 1)
	return id, nil
}
