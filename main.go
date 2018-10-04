package main

import (
	"github.com/pippozq/pushgateway/global"
	"github.com/pippozq/pushgateway/routes"
)

func main() {
	global.Config.Server.Serve(routes.RootRouter)
}
