package main

import (
	"flag"
	"fmt"
	//"../gedis-benchmark/cacheClient"
	"github.com/gauge2009/micro-golang/gedis-benchmark/cacheClient"
)

func main() {
	server := flag.String("h", "localhost", "cache server address")
	op := flag.String("c", "get", "command, could be get/set/del")
	key := flag.String("k", "", "key")
	value := flag.String("v", "", "value")
	flag.Parse()
	client := cacheClient.New("tcp", *server)
	cmd := &cacheClient.Cmd{*op, *key, *value, nil}
	//./client -c set -k info1 -v  '{  \"result\": \"success\",   \"error\": null }'
	//./client -c get -k info1
	client.Run(cmd)
	if cmd.Error != nil {
		fmt.Println("error:", cmd.Error)
	} else {
		fmt.Println(cmd.Value)
	}
}
