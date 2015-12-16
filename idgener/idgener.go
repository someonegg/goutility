// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package idgener can generate ids.
package idgener

import (
	"golang.org/x/net/context"
	"io"
)

type IDGener interface {
	io.Closer
	// The first is 1.
	GenID(ctx context.Context) (id int64, err error)
}
