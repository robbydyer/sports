// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin,metal

package driver

import (
	"github.com/robbydyer/exp/shiny/driver/mtldriver"
	"github.com/robbydyer/exp/shiny/screen"
)

func main(f func(screen.Screen)) {
	mtldriver.Main(f)
}
