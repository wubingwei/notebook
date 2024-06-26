package main

import (
	"bufio"
	"log"
	"os"
	"reflect"
	"runtime"
	"strconv"
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

/*
goos: darwin
goarch: arm64
pkg: github.com/wubingwei/notebook/filter/code
BenchmarkBloomTest-8   	 4907282	       257.6 ns/op	     126 B/op	       3 allocs/op
*/
func BenchmarkBloomTest(b *testing.B) {
	f := NewBloomFilter(1e8, 0.01)
	var testObject int64 = 5e7
	for i := int64(1); i < testObject; i += 1 {
		f.BF.AddString("wubingwei " + strconv.FormatInt(i, 10))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		f.BF.Test([]byte("wubingwei " + strconv.FormatInt(int64(i), 10)))
	}
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

func TestProductionBloom(t *testing.T) {
	Convey("BloomFilter\n", t, func() {
		n, errRate := uint(10000), 0.01

		bf := NewBloomFilter(n, errRate)

		m, f := bloom.EstimateParameters(n, errRate)
		t.Logf("object number = %d", n)
		t.Logf("errRate = %.2f", errRate)
		t.Logf("length of bitset = %d", m)
		t.Logf("hash function number = %d", f)
		t.Logf("size of bitset = %d M-Bytes", GetSliceMemorySize(make([]uint64, int64(m>>6)))>>20)

		file, err := os.Open("../_file/test.csv")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// 创建一个 Scanner 对象，用于逐行读取文件内容
		scanner := bufio.NewScanner(file)

		start := time.Now()
		// 逐行读取文件内容
		var num int
		for scanner.Scan() {
			line := scanner.Text()
			num += 1
			if num >= int(n) {
				break
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

		start = time.Now()

		var errNumTest int64
		var testObject int64 = 100000
		for i := int64(1); i < testObject; i += 1 {
			if g.TestString("wubingwei " + strconv.FormatInt(i, 10)) {
				errNumTest += 1
			}
		}
		t.Logf("Test True: %s, actual: %v", "103a5d8a882bfed1742653bb2eea81bc,com.inletfilter.test,rewarded_video", g.TestString("103a5d8a882bfed1742653bb2eea81bc,com.inletfilter.test,rewarded_video"))
		t.Logf("Test True: %s, actual: %v", "e73bc49a8e8dccf488301863b3c0364e", g.TestString("e73bc49a8e8dccf488301863b3c0364e"))

		t.Logf("Test Error Rate: %f, errNum = %d, testObject = %d\n, usedTime = %d us", float64(errNumTest)/float64(testObject), errNumTest, testObject, time.Since(start).Microseconds())

		t.Logf("Test wubingwei should be false, actual = %v", g.TestString("wubingwei"))

		start = time.Now()
		runtime.GC()
		t.Logf("GC used %d ms", time.Since(start).Milliseconds())
	})
}
