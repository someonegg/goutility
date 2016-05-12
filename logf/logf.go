// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logf can control log-file rotation of stdlog.
//
// The simplest way to use logf is:
//   package main
//
//   import (
//   	"github.com/someonegg/goutility/logf"
//   )
//
//   func main() {
//      ...
//   	err = logf.SetOutput(logFilePath)
//   	...
//   }
//
// A simple rotation sh is like:
//   nowtime=$(date "+%Y%m%d%H%M%S")
//   mv -f xxx.log ${nowtime}.xxx.log
//   kill -s SIGUSR1 pid_of_xxx
//
package logf

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	locker sync.Mutex
	logS   string
	logF   *os.File
)

func SetOutput(path string) error {
	file, err := os.OpenFile(path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	log.SetOutput(file)

	locker.Lock()
	defer locker.Unlock()

	if logF != nil {
		logF.Close()
	}

	logS = path
	logF = file

	return nil
}

func init() {
	go logSig()
}

func logSig() {
	defer func() { recover() }()

	// SIGUSR1 to reload log.
	rC := make(chan os.Signal, 1)
	signal.Notify(rC, syscall.SIGUSR1)

	for {
		select {
		case <-rC:
			locker.Lock()
			path := logS
			locker.Unlock()
			if len(path) > 0 {
				SetOutput(path)
			}
		}
	}
}
