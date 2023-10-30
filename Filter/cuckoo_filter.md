# 布谷鸟过滤器(Cuckoo Filter）

## 主要作用
主要用于检索一个元素是否在一个集合中

## 优点
- 它支持动态添加和删除项；
- 它提供了比传统的布隆过滤器更高的查找性能，即使当其接近满载（例如，95%的空间已被使用）；
- 它比诸如商过滤器等替代品更容易实现；
- 在许多实际应用中，如果目标假阳性率ε小于3%，则它使用的空间小于布隆过滤器。

## 结构
```GO
fliter 结构定义 [][4]byte
其中 byte 表示的是数据的指纹，定义为 fingerprint
每个数据有两个 buckt 的 index1 and index2。
首先插入index1 的 bucket，在 [4]byte 按顺序循环找到 空位，如没有，插入index1 的 bucket，依旧在 [4]byte 按顺序循环找到 空位。
```


## 开源实现
Go: https://github.com/panmari/cuckoofilter

### 关键代码


## 测试
```Go
CuckooProduction

cuckoo_test.go:86: object number = 200000000
cuckoo_test.go:87: size of cuckooFilter = 296 M-Bytes
cuckoo_test.go:106: total object: 89142232
cuckoo_test.go:108: load elapsed = 14.087430 s
cuckoo_test.go:111: wd: /Users/David/Golang/src/github.com/wubingwei/notebook/Filter/code
cuckoo_test.go:115: File created: ../_file/cuckoo_filter_binary_file
cuckoo_test.go:122: bytesWritten = 268435480 Bytes
cuckoo_test.go:147: Test Error Rate: 0.009900, errNum = 99, testObject = 10000
cuckoo_test.go:149: Test wubingwei should be false, actual = false
```