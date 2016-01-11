// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pidf can generate pid file.
package pidf

import (
	"os"
	"strconv"
)

type PidFile struct {
	path string
	Pid  int
}

func New(path string) *PidFile {
	t := &PidFile{path, os.Getpid()}

	f, err := os.OpenFile(path,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return t
	}
	defer f.Close()

	_, err = f.WriteString(strconv.Itoa(t.Pid))

	return t
}

func (pf *PidFile) Close() error {
	return os.Remove(pf.path)
}
