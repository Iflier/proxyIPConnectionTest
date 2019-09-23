package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

var lock = new(sync.Mutex)

type ProxyIP struct {
	HttpType string
	Ip       string
}

var redisOptions = redis.Options{
	Network:    "tcp",
	Addr:       "localhost:6379",
	DB:         0,
	Password:   "123,gslw",
	MaxRetries: 5,
}

func testProxyIP(proxyStr string) bool {
	var returnVal bool = false
	var conn, err = net.DialTimeout("tcp4", proxyStr, time.Second*time.Duration(60))
	if err != nil {
		fmt.Println("Connection test failed !")
		returnVal = false
	} else {
		// 可连通的连接，需要主动关闭。没有创建连接的就不用了
		conn.Close()
		fmt.Println("Connection test ok")
		returnVal = true
	}
	return returnVal
}

func testAndAddProxyIP(ipList *list.List, cache *redis.Client) {
	var testResult bool
	for elem := ipList.Front(); elem != nil; elem = elem.Next() {
		testResult = testProxyIP(elem.Value.(string))
		if testResult != true {
			cache.SRem("HTTP", elem.Value.(string))
		}
	}
}

func main() {
	var elemList = list.New()
	var stringSlicePointer *redis.StringSliceCmd
	var intCmdPointer *redis.IntCmd
	// http 和 https 类型的代理。也代表着redis中的两种集合
	var validHttpType = [...]string{"HTTP", "HTTPS"}
	var structInstance *ProxyIP = new(ProxyIP)
	var cache = redis.NewClient(&redisOptions)
	var file, err = os.Open("proxy.json")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	var fileReader = bufio.NewReader(file)
	for {
		if content, err := fileReader.ReadBytes('\n'); err == nil {
			json.Unmarshal(content, structInstance)
			cache.SAdd(strings.ToUpper(structInstance.HttpType), structInstance.Ip)
		} else {
			break
		}
	}
	for _, httpType := range validHttpType {
		intCmdPointer = cache.SCard(httpType)
		// 仅当集合元素个数大于 0 时
		if intCmdPointer.Val() > 0 {
			stringSlicePointer = cache.SMembers(httpType)
			for _, value := range stringSlicePointer.Val() {
				elemList.PushBack(value)
			}
		}
	}
	testAndAddProxyIP(elemList, cache)
}
