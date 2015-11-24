// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statutil

type Price struct {
	count      uint64
	cycle      []int64
	cycleTotal int64
	average    int64
}

func NewPrice(cycleSize int) *Price {
	return &Price{
		cycle:   make([]int64, cycleSize, cycleSize),
		count:   0,
		average: 0,
	}
}

func (p *Price) Update(sample int64) {
	N := uint64(len(p.cycle))
	i := p.count % N
	p.count++

	p.cycleTotal -= p.cycle[i]
	p.cycleTotal += sample
	p.cycle[i] = sample

	n := p.count
	if n > N {
		n = N
	}
	p.average = p.cycleTotal / int64(n)
}

func (p *Price) Average() int64 {
	return p.average
}
