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

func genListElem(ch chan string, wg *sync.WaitGroup, ipList *list.List) {
	// 通过 chan 向各个 go routines 传递列表元素
	var next *list.Element
	wg.Add(1)
	if ipList.Len() > 0 {
		for elem := ipList.Front(); elem != nil; elem = next {
			ch <- elem.Value.(string)
			next = elem.Next()
			ipList.Remove(elem)
		}
	}
	for i := 0; i < 10; i++ {
		ch <- ""
		time.Sleep(10 * time.Millisecond)
	}
	wg.Done()
}

func testAvailableProxyIP(proxyStr string) bool {
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

func addProxyIP(elemChan chan string, wg *sync.WaitGroup, cache *redis.Client) {
	var testResult bool
	for testIP := range elemChan {
		if testIP != "" {
			testResult = testAvailableProxyIP(testIP)
			if testResult != true {
				cache.SRem("HTTP", testIP)
				fmt.Printf("Remove element: %v\n", testIP)
			}
		} else {
			break
		}
	}
	// main函数可以退出了
	wg.Done()
}

func main() {
	var elemList = list.New()
	var elemChan = make(chan string)
	var waitGroup = new(sync.WaitGroup)
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
			fmt.Printf("%v elements in %v\n", intCmdPointer.Val(), httpType)
			stringSlicePointer = cache.SMembers(httpType)
			for _, value := range stringSlicePointer.Val() {
				elemList.PushBack(value)
			}
		}
	}
	go genListElem(elemChan, waitGroup, elemList)
	for i := 0; i < 10; i++ {
		fmt.Printf("i = %v\n", i)
		waitGroup.Add(1)
		go addProxyIP(elemChan, waitGroup, cache)
	}
	// 等待所有的go routines完成
	waitGroup.Wait()
	fmt.Println("Done.")
}
