# 永不阻塞, 无限缓存的 channel

Ref: 实现无限缓存的channel | 鸟窝 https://colobu.com/2021/05/11/unbounded-channel-in-go/

*forked from smallnest/chanx*

## 变动

- 增加初始化可选参数 `maxBufCapacity` 
  - 用于限定最大缓存数量, 将无限缓存变为永不阻塞的有限缓存队列, 超过限制丢弃数据
- 增加数据丢弃时回调方法: `c.SetOnDiscards(func(T))`
  - 用于数据达到限定的最大缓存数量并丢弃时, 将数据传给回调方法处理(无限缓存无效)
- 增加动态调整最大缓存数量方法: `c.SetMaxCapacity(0)`
  - 值为 0 时恢复无限缓存, 返回值为 0 或当前最大缓存限制数(含初始容量)
- 增加一些计数方法: `c.BufCapacity()` `c.MaxBufSize()` `c.Discards()`

## 使用

- 通用: [unbounded_chan_test.go](unbounded_chan_test.go)
- 泛型: [unbounded_chanof_test.go](unbounded_chanof_test.go)

### 安装

```go
go get github.com/fufuok/chanx
```

### 永不阻塞, 无限缓存的 Channel

```go
package main

import (
	"fmt"

	"github.com/fufuok/chanx"
)

func main() {
	ch := chanx.NewUnboundedChanOf[int](10)
	// or ch := chanx.NewUnboundedChanSize[int](10, 200, 1000)

	go func() {
		for i := 0; i < 100; i++ {
			ch.In <- i // send values
		}
		close(ch.In) // close In channel
	}()

	for v := range ch.Out { // read values
		fmt.Println(v + 1)
	}
}
```

### 永不阻塞, 带缓存上限的 Channel

```go
package main

import (
	"fmt"

	"github.com/fufuok/chanx"
)

func main() {
	// 可选参数, 缓冲上限
	const maxBufCapacity = 10
	ch := chanx.NewUnboundedChanOf[int](10, maxBufCapacity)
	// or
	// ch := chanx.NewUnboundedChanSize[int](10, 10, 10, maxBufCapacity)

	// 有缓冲上限时, 可选设置数据丢弃时回调
	ch.SetOnDiscards(func(v int) {
		fmt.Println("discard: ", v)
	})

	go func() {
		for i := 0; i < 100; i++ {
			ch.In <- i // send values
		}
		close(ch.In) // close In channel
	}()

	// time.Sleep(time.Second)

	for v := range ch.Out { // read values
		fmt.Println(v + 0)
	}
}
```



# chanx

Unbounded chan.

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/smallnest/chanx?status.png)](http://godoc.org/github.com/smallnest/chanx)  [![travis](https://travis-ci.org/smallnest/chanx.svg?branch=main)](https://travis-ci.org/smallnest/chanx) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/chanx)](https://goreportcard.com/report/github.com/smallnest/chanx) [![coveralls](https://coveralls.io/repos/smallnest/chanx/badge.svg?branch=main&service=github)](https://coveralls.io/github/smallnest/chanx?branch=main) 

Refer to the below articles and issues:
1. https://github.com/golang/go/issues/20352
2. https://stackoverflow.com/questions/41906146/why-go-channels-limit-the-buffer-size
3. https://medium.com/capital-one-tech/building-an-unbounded-channel-in-go-789e175cd2cd
4. https://erikwinter.nl/articles/2020/channel-with-infinite-buffer-in-golang/

