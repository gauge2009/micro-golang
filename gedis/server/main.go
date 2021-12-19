package main

import (
	//"./cache"
	//"./http"
	//"./tcp"
	"github.com/gauge2009/micro-golang/gedis/server/cache"
	"github.com/gauge2009/micro-golang/gedis/server/http"
	"github.com/gauge2009/micro-golang/gedis/server/tcp"
)

func main() {
	ca := cache.New("inmemory")
	go tcp.New(ca).Listen()
	http.New(ca).Listen()
}
