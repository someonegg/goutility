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

type ContextHandler interface {
	ContextServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// HttpService is a wrapper of http.Server.
type HttpService struct {
	err     error
	quitCtx context.Context
	quitF   context.CancelFunc
	stopD   chanutil.DoneChan

	l   *net.TCPListener
	h   ContextHandler
	srv *http.Server

	reqWG sync.WaitGroup
}

// NewHttpService is a short cut to use NewHttpServiceEx.
func NewHttpService(l *net.TCPListener, h http.Handler,
	maxConcurrent int) *HttpService {

	return NewHttpServiceEx(l, NewMaxConcurrentHandler(NewHttpHandler(h),
		maxConcurrent, DefaultHesitateTime, DefaultMaxConcurrentNotifier))
}

func NewHttpServiceEx(l *net.TCPListener, h ContextHandler) *HttpService {
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

	return s
}

func (s *HttpService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.reqWG.Add(1)
	defer s.reqWG.Done()
	s.h.ContextServeHTTP(s.quitCtx, w, r)
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

	s.quitF()
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

type httpHandler struct {
	oh http.Handler
}

// The http handler type is an adapter to allow the use of
// ordinary http.Handler as ContextHandler.
func NewHttpHandler(oh http.Handler) ContextHandler {
	return httpHandler{oh}
}

func (h httpHandler) ContextServeHTTP(ctx context.Context,
	w http.ResponseWriter, r *http.Request) {

	h.oh.ServeHTTP(w, r)
}

type maxConcurrentHandler struct {
	oh           ContextHandler
	concur       chanutil.Semaphore
	hesitateTime time.Duration
	notifier     MaxConcurrentNotifier
}

// The maxconcurrent handler type is a middleware that can limit the
// maximum number of concurrent access.
// if maxConcurrent == 0, no limit on concurrency.
// if hesitateTime == 0, use DefaultHesitateTime.
func NewMaxConcurrentHandler(oh ContextHandler,
	maxConcurrent int, hesitateTime time.Duration,
	notifier MaxConcurrentNotifier) ContextHandler {

	if maxConcurrent <= 0 {
		return oh
	}
	if hesitateTime <= 0 {
		hesitateTime = DefaultHesitateTime
	}

	return &maxConcurrentHandler{
		oh:           oh,
		concur:       chanutil.NewSemaphore(maxConcurrent),
		hesitateTime: hesitateTime,
		notifier:     notifier,
	}
}

const (
	acquire_OK      int = 0
	acquire_CtxDone int = 1
	acquire_Timeout int = 2
)

func (h *maxConcurrentHandler) acquireConn(ctx context.Context) int {
	select {
	case <-ctx.Done():
		return acquire_CtxDone
	// Acquire
	case h.concur <- struct{}{}:
		return acquire_OK
	case <-time.After(h.hesitateTime):
		return acquire_Timeout
	}
}

func (h *maxConcurrentHandler) releaseConn() {
	<-h.concur
}

func (h *maxConcurrentHandler) ContextServeHTTP(ctx context.Context,
	w http.ResponseWriter, r *http.Request) {

	ret := h.acquireConn(ctx)
	switch ret {
	case acquire_CtxDone:
		h.notifier.OnContextDone(w, r)
		return
	case acquire_Timeout:
		h.notifier.OnConcurrentLimit(w, r)
		return
	}
	defer h.releaseConn()

	h.oh.ContextServeHTTP(ctx, w, r)
}

const DefaultHesitateTime = 50 * time.Millisecond

type MaxConcurrentNotifier interface {
	OnContextDone(w http.ResponseWriter, r *http.Request)
	OnConcurrentLimit(w http.ResponseWriter, r *http.Request)
}

type defaultMaxConcurrentNotifier struct{}

func (n defaultMaxConcurrentNotifier) OnContextDone(
	w http.ResponseWriter, r *http.Request) {

	http.Error(w, "Service Maintenance", http.StatusServiceUnavailable)
}

func (n defaultMaxConcurrentNotifier) OnConcurrentLimit(
	w http.ResponseWriter, r *http.Request) {

	http.Error(w, "Service Busy", http.StatusServiceUnavailable)
}

var DefaultMaxConcurrentNotifier MaxConcurrentNotifier = defaultMaxConcurrentNotifier{}
