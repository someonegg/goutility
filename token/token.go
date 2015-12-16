// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package token can generate token.
package token

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
)

func Generate() string {
	var r [64]byte
	rand.Read(r[0:])
	return fmt.Sprintf("%x", md5.Sum(r[0:]))
}
