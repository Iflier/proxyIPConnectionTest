package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-redis/redis"
)

type ProxyIP struct {
	HttpType string
	Ip       string
}

var redisOptions = redis.Options{
	Network:    "tcp",
	Addr:       "localhost:6379",
	DB:         0,
	Password:   "******",
	MaxRetries: 5,
}

func testProxyIP(proxyStr string) bool {
	var returnVal bool = false
	var conn, err = net.DialTimeout("tcp4", proxyStr, time.Second*time.Duration(60))
	if err != nil {
		fmt.Println("Connection test failed !")
		returnVal = false
	} else {
		conn.Close()
		fmt.Println("Connection test ok")
		returnVal = true
	}
	return returnVal
}

func main() {
	var testResult bool
	var structInstance *ProxyIP = new(ProxyIP)
	var cache = redis.NewClient(&redisOptions)
	var file, err = os.Open("proxy.json")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	// var cmdPoint = cache.SCard("HTTP")
	// fmt.Println(cmdPoint.Val())
	var fileReader = bufio.NewReader(file)
	for {
		if content, err := fileReader.ReadBytes('\n'); err == nil {
			json.Unmarshal(content, structInstance)
			testResult = testProxyIP(structInstance.Ip)
			if testResult {
				cache.SAdd(structInstance.HttpType, structInstance.Ip)
			}
		} else {
			break
		}
	}
}
