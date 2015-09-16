// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package poolutil

import (
	"sync"
)

var bufTypes = [...]int{
	16, 32, 48, 64, 80, 96, 112,
	128, 160, 192, 224,
	256, 320, 384, 448,
	512, 640, 768, 896,
	1024,
}

const bufTypeNum = len(bufTypes)

var bufPools [bufTypeNum]sync.Pool

func init() {
	for i := 0; i < bufTypeNum; i++ {
		l := bufTypes[i]
		bufPools[i].New = func() interface{} {
			return make([]byte, l, l)
		}
	}
}

func BufGet(size int) []byte {
	if size == 0 {
		return nil
	}

	if size <= bufTypes[bufTypeNum-1] {

		for i := 0; i < bufTypeNum; i++ {
			l := bufTypes[i]
			if size <= l {
				b := bufPools[i].Get().([]byte)
				return b[0:size]
			}
		}
	}

	return make([]byte, size, size)
}

func BufPut(b []byte) {
	size := cap(b)

	if size == 0 {
		return
	}

	if size <= bufTypes[bufTypeNum-1] {

		for i := 0; i < bufTypeNum; i++ {
			l := bufTypes[i]
			if size <= l {
				if size == l {
					bufPools[i].Put(b[0:size])
				}
				return
			}
		}
	}

	return
}
