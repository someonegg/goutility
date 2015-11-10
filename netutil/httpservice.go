// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netutil

import (
	"errors"
	"github.com/someonegg/goutility/chanutil"
	"golang.org/x/net/context"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	ErrUnknownPanic = errors.New("unknown panic")
)

// HttpService is a wrapper of http.Server.
type HttpService struct {
	err     error
	quitCtx context.Context
	quitF   context.CancelFunc
	stopD   chanutil.DoneChan

	l   *net.TCPListener
	h   http.Handler
	srv *http.Server

	concur chanutil.Semaphore
	reqWG  sync.WaitGroup
}

// if maxConcurrent == 0, no limit on concurrency.
func NewHttpService(l *net.TCPListener, h http.Handler, maxConcurrent int) *HttpService {
	s := &HttpService{}

	s.quitCtx, s.quitF = context.WithCancel(context.Background())
	s.stopD = chanutil.NewDoneChan()
	s.l = l
	s.h = h
	s.srv = &http.Server{
		Addr:           s.l.Addr().String(),
		Handler:        s,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if maxConcurrent > 0 {
		s.concur = chanutil.NewSemaphore(maxConcurrent)
	}

	return s
}

const hesitateTime = 50 * time.Millisecond

var (
	acquire_OK      int = 0
	acquire_Quit    int = 1
	acquire_Timeout int = 2
)

func (s *HttpService) acquireConn() int {
	if s.concur == nil {
		return acquire_OK
	}

	select {
	case <-s.quitCtx.Done():
		return acquire_Quit
	// Acquire
	case s.concur <- struct{}{}:
		return acquire_OK
	case <-time.After(hesitateTime):
		return acquire_Timeout
	}
}

func (s *HttpService) releaseConn() {
	if s.concur == nil {
		return
	}

	<-s.concur
}

func (s *HttpService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ret := s.acquireConn()
	switch ret {
	case acquire_Quit:
		http.Error(w, "Service Unavailable!", http.StatusServiceUnavailable)
		return
	case acquire_Timeout:
		http.Error(w, "Service Busy!", http.StatusRequestTimeout)
		return
	}
	defer s.releaseConn()

	s.reqWG.Add(1)
	defer s.reqWG.Done()
	s.h.ServeHTTP(w, r)
}

func (s *HttpService) Start() {
	go s.serve()
}

func (s *HttpService) serve() {
	defer s.ending()

	s.err = s.srv.Serve(TcpKeepAliveListener{s.l})
}

func (s *HttpService) ending() {
	if e := recover(); e != nil {
		switch v := e.(type) {
		case error:
			s.err = v
		default:
			s.err = ErrUnknownPanic
		}
	}

	s.stopD.SetDone()
}

func (s *HttpService) Err() error {
	return s.err
}

func (s *HttpService) Stop() {
	s.srv.SetKeepAlivesEnabled(false)
	s.quitF()
	s.l.Close()
}

func (s *HttpService) StopD() chanutil.DoneChanR {
	return s.stopD.R()
}

func (s *HttpService) Stopped() bool {
	return s.stopD.R().Done()
}

func (s *HttpService) WaitRequests() {
	s.reqWG.Wait()
}

func (s *HttpService) QuitCtx() context.Context {
	return s.quitCtx
}
