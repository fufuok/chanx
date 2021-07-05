# 无限缓存的 channel

Ref: 实现无限缓存的channel | 鸟窝 https://colobu.com/2021/05/11/unbounded-channel-in-go/

*forked from smallnest/chanx*

## 变动

- 增加初始化可选参数 `maxBufCapacity` 用于限定无限缓存为最大缓存数量, 超过限制丢弃数据
- 增加动态调整最大缓存数量方法: `c.SetMaxCapacity(0)`, 值为 0 时恢复无限, 返回 0 或当前最大缓存限制数(含初始容量)
- 增加一些计数方法: `c.BufCapacity()` `c.MaxBufSize()` `c.Discards()`

# chanx

Unbounded chan.

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/smallnest/chanx?status.png)](http://godoc.org/github.com/smallnest/chanx)  [![travis](https://travis-ci.org/smallnest/chanx.svg?branch=main)](https://travis-ci.org/smallnest/chanx) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/chanx)](https://goreportcard.com/report/github.com/smallnest/chanx) [![coveralls](https://coveralls.io/repos/smallnest/chanx/badge.svg?branch=main&service=github)](https://coveralls.io/github/smallnest/chanx?branch=main) 

Refer to the below articles and issues:
1. https://github.com/golang/go/issues/20352
2. https://stackoverflow.com/questions/41906146/why-go-channels-limit-the-buffer-size
3. https://medium.com/capital-one-tech/building-an-unbounded-channel-in-go-789e175cd2cd
4. https://erikwinter.nl/articles/2020/channel-with-infinite-buffer-in-golang/

## Usage

```go
go get github.com/fufuok/chanx
```

```go
ch := NewUnboundedChan(1000)
// or ch := NewUnboundedChanSize(10,200,1000)

go func() {
    for ...... {
        ...
        ch.In <- ... // send values
        ...
    }

    close(ch.In) // close In channel
}()


for v := range ch.Out { // read values
    fmt.Println(v)
}
```