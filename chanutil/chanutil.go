// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chanutil contains a set of utilities to facilitate
// the management and use of chan.
package chanutil

import (
	"errors"
	"sync"
)

var (
	ErrInvalidArgs = errors.New("invalid arguments")
)

// Chan id.
type ChanID int64

// Chanmap can create and index chans. Every chan created by
// chanmap has a unique id, so you can get them by the id.
//
// Multiple goroutines can invoke methods on a Chanmap simultaneously.
type Chanmap struct {
	chans       map[ChanID]chan interface{}
	chanLock    sync.RWMutex
	nextID      ChanID
	defChanSize int
}

func NewChanmap(defChanSize int) *Chanmap {
	return &Chanmap{
		chans:       make(map[ChanID]chan interface{}),
		nextID:      1,
		defChanSize: defChanSize,
	}
}

func (m *Chanmap) New() (id ChanID, c chan interface{}) {
	return m.NewF(m.defChanSize)
}

func (m *Chanmap) NewF(size int) (id ChanID, c chan interface{}) {
	m.chanLock.Lock()
	defer m.chanLock.Unlock()
	id = m.nextID
	c = make(chan interface{}, size)
	m.chans[id] = c
	m.nextID += 1
	return
}

func (m *Chanmap) Get(id ChanID) chan interface{} {
	m.chanLock.RLock()
	defer m.chanLock.RUnlock()
	return m.chans[id]
}

func (m *Chanmap) Remove(id ChanID) {
	m.chanLock.Lock()
	defer m.chanLock.Unlock()
	delete(m.chans, id)
}

var defChanmap = NewChanmap(1)

// Call on the default Chanmap
func New() (id ChanID, c chan interface{}) {
	return defChanmap.New()
}

// Call on the default Chanmap
func NewF(size int) (id ChanID, c chan interface{}) {
	return defChanmap.NewF(size)
}

// Call on the default Chanmap
func Get(id ChanID) chan interface{} {
	return defChanmap.Get(id)
}

// Call on the default Chanmap
func Remove(id ChanID) {
	defChanmap.Remove(id)
}

// Auto reset event.
type Event chan struct{}

func NewEvent() Event {
	return make(chan struct{}, 1)
}

func (e Event) Set() {
	select {
	case e <- struct{}{}:
	default:
	}
}

func (e Event) R() EventR {
	return EventR((chan struct{})(e))
}

// You can determine whether event setted through EventR.
type EventR <-chan struct{}

// You can notify something done through DoneChan.SetDone().
type DoneChan chan struct{}

func NewDoneChan() DoneChan {
	return make(chan struct{})
}

func (d DoneChan) SetDone() {
	defer func() { recover() }()
	select {
	case <-d:
	default:
		close(d)
	}
}

func (d DoneChan) R() DoneChanR {
	return DoneChanR((chan struct{})(d))
}

// You can determine whether something done through DoneChanR.Done().
type DoneChanR <-chan struct{}

func (d DoneChanR) Done() bool {
	select {
	case <-d:
		return true
	default:
		return false
	}
}

// Semaphore can be used to limit access to multiple resources.
type Semaphore chan struct{}

func NewSemaphore(n int) Semaphore {
	if n <= 0 {
		panic(ErrInvalidArgs)
	}
	return make(chan struct{}, n)
}

// Acquire n resources.
//
// s <- e
func (s Semaphore) Acquire(n int) {
	if n > cap(s) {
		panic(ErrInvalidArgs)
	}
	e := struct{}{}
	for i := 0; i < n; i++ {
		s <- e
	}
}

// Release n resources.
//
// e := <-s
func (s Semaphore) Release(n int) {
	if n > cap(s) {
		panic(ErrInvalidArgs)
	}
	for i := 0; i < n; i++ {
		<-s
	}
}
