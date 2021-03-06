package main

import (
	"fmt"
	"time"
)

// Counter 代表计数器的类型。
type Counter struct {
	count int
}

func (counter *Counter) String() string {
	return fmt.Sprintf("{count=:%d}", counter.count)
}

var mapChan = make(chan map[string]Counter, 1)

func main() {
	syncChan := make(chan struct{}, 2)
	go func() { // 用于演示接收操作。
		for {
			if elem, ok := <-mapChan; ok {
				//counter是一个Counter结构体类型
				counter := elem["count"]
				counter.count++
			} else {
				break
			}
		}
		fmt.Println("Stopped. [receiver]")
		syncChan <- struct{}{}
	}()
	go func() { // 用于演示发送操作。
		countMap := map[string]Counter{
			"count": Counter{},
		}
		for i := 0; i < 5; i++ {
			//每次都发送的同一个map，但是K,V这里的V是结构体，传递过去是值
			//map虽然是引用，但是map内部结构count是一个普通结构体类型，所以每次传过去的count都是一个复制后的结构体
			mapChan <- countMap
			time.Sleep(time.Millisecond)
			fmt.Printf("The count map: %v. [sender]\n", countMap)
		}
		close(mapChan)
		syncChan <- struct{}{}
	}()
	<-syncChan
	<-syncChan
}
//演示结果
//The count map: map[count:{0}]. [sender]
//The count map: map[count:{0}]. [sender]
//The count map: map[count:{0}]. [sender]
//The count map: map[count:{0}]. [sender]
//The count map: map[count:{0}]. [sender]
//Stopped. [receiver]