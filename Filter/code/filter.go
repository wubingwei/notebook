package main

type Filter interface {
	Add(string)       // 添加数据
	Test(string) bool // 验证数据
}
