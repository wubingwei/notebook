package main

import (
	"bufio"
	"log"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/bits-and-blooms/bloom"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBasic(t *testing.T) {
	Convey("Basic", t, func() {
		f := NewBloomFilter(1000, 0.001)
		n1, n2, n3 := "Bess", "David", "Emma"
		f.Add(n1)
		f.Add(n3)
		n1b := f.Test(n1)
		n2b := f.Test(n2)
		n3b := f.Test(n3)
		So(n1b, ShouldBeTrue)
		So(n2b, ShouldBeFalse)
		So(n3b, ShouldBeTrue)
	})
}

func GetSliceMemorySize(slice interface{}) uintptr {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		panic("Input is not a slice")
	}

	elemSize := unsafe.Sizeof(sliceValue.Index(0).Interface())
	headerSize := unsafe.Sizeof(reflect.SliceHeader{})
	length := uintptr(sliceValue.Len())
	capacity := uintptr(sliceValue.Cap())

	return length*elemSize + headerSize + capacity*elemSize
}

func TestProduct(t *testing.T) {
	Convey("BloomFilter\n", t, func() {
		n, errRate := uint(1e9), 0.01

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		t.Logf("Before System memory size: %d M-bytes\n", memStats.Sys>>20)
		bf := NewBloomFilter(n, errRate)

		runtime.ReadMemStats(&memStats)
		t.Logf("After1 System memory size: %d M-bytes\n", memStats.Sys>>20)

		m, f := bloom.EstimateParameters(n, errRate)
		t.Logf("object number = %d", n)
		t.Logf("errRate = %.2f", errRate)
		t.Logf("length of bitset = %d", m)
		t.Logf("hash function number = %d", f)
		//t.Logf("size of bitset = %d G-Bytes", GetSliceMemorySize(make([]uint64, int64(m)))>>30)

		file, err := os.Open("../_file/part-00000-2c9152b7-c072-479a-97e0-10c26c90bb38-c000.csv")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// 创建一个 Scanner 对象，用于逐行读取文件内容
		scanner := bufio.NewScanner(file)

		start := time.Now()
		// 逐行读取文件内容
		var num int
		var testObject []string
		for scanner.Scan() {
			line := scanner.Text()
			num += 1
			if num%100 == 0 {
				testObject = append(testObject, line)
			}
			bf.Add(line)
		}
		t.Logf("total object: %d", num)

		t.Logf("load elapsed = %f s", time.Since(start).Seconds())

		wd, _ := os.Getwd()
		t.Logf("wd: %s", wd)

		wFile, _ := os.Create("../_file/bloom_filter_binary_file")
		defer wFile.Close()
		t.Log("File created:", wFile.Name())

		w := bufio.NewWriter(wFile)
		bytesWritten, err := bf.BF.WriteTo(w)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("bytesWritten = %d M-Bytes", bytesWritten>>20)

		rFile, _ := os.Open("../_file/bloom_filter_binary_file")
		r := bufio.NewReader(rFile)
		var g bloom.BloomFilter
		bytesRead, err := g.ReadFrom(r)
		if err != nil {
			t.Fatal(err.Error())
		}
		if bytesRead != bytesWritten {
			t.Errorf("read unexpected number of bytes %d != %d", bytesRead, bytesWritten)
		}

		// add 的元素一定存在
		var errNum int
		for _, o := range testObject {
			if !g.TestString(o) {
				errNum += 1
			}
		}
		t.Logf("Test Error Rate: %f, errNum = %d, testObject = %d\n", float64(errNum/len(testObject)), errNum, len(testObject))

		t.Logf("Test wubingwei should not false, actual = %v", g.TestString("wubingwei"))
		runtime.ReadMemStats(&memStats)
		t.Logf("After2 System memory size: %d M-bytes\n", memStats.Sys>>20)
	})
}
