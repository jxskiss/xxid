# XXID Unique ID Generator

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/jxskiss/xxid/v2) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/jxskiss/xxid/master/LICENSE)

Package xxid is a specific kind of unique id generator, it was originally forked from [rs/xid][].

The v2 version has been totally redesigned and rewritten.

[rs/xid]: https://github.com/rs/xid/

## Introduction

TODO

(mention clock adjusted back and leap second)

(mention https://github.com/segmentio/ksuid)
(mention https://github.com/rs/xid)

(mention security concerns)

## Install

```shell
go get github.com/jxskiss/xxid/v2@latest
```

## Usage

```go
id := xxid.New()
fmt.Printf("%d %s %s\n", id.Short(), id.Base62(), id.String())
// 107306765558351289 0MTmSIz6YnbzdVsgK5S7SE 20211120092140634056218b67c800abe705b9

ip := net.ParseIP("10.9.8.7")
gen := xxid.NewGenerator().UseIPv4(ip).UsePort(8888).UseFlag(123)
id = gen.New()
fmt.Println(id.Time(), id.IP(), id.Port(), id.Flag())
// 2021-11-20 09:21:40.634 +0800 CST 10.9.8.7 8888 123
```
