package main

import "github.com/bits-and-blooms/bloom"

type BloomFilter struct {
	BF *bloom.BloomFilter
}

func NewBloomFilter(n uint, fp float64) *BloomFilter {
	f := bloom.NewWithEstimates(n, fp)
	return &BloomFilter{
		BF: f,
	}
}

func (bf *BloomFilter) Add(s string) {
	bf.BF.AddString(s)
}

func (bf *BloomFilter) Test(s string) bool {
	return bf.BF.TestString(s)
}
