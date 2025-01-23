package main

import (
	"fmt"
	"github.com/duke-git/lancet/v2/netutil"
)

func main() {
	ping := netutil.IsPingConnected("127.0.0.1")
	fmt.Println(ping)

	ping = netutil.IsPingConnected("10.10.10.20")
	fmt.Println(ping)
}
