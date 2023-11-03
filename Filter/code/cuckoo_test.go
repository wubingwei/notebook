package main

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
	"unsafe"

	cuckoo "github.com/seiflotfy/cuckoofilter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCuckooBasic(t *testing.T) {
	Convey("CuckooBasic", t, func() {
		cf := cuckoo.NewFilter(1000)

		var testObject int64 = 100
		for i := int64(1); i < testObject; i += 1 {
			cf.InsertUnique([]byte("wubingwei " + strconv.FormatInt(i, 10)))
		}

		So(cf.Lookup([]byte("wubingwei 0")), ShouldBeFalse)
		So(cf.Lookup([]byte("wubingwei 1")), ShouldBeTrue)

		So(cf.Count(), ShouldEqual, 99)

		// Delete a string (and it a miss)
		cf.Delete([]byte("wubingwei 1"))

		So(cf.Lookup([]byte("wubingwei 1")), ShouldBeFalse)
		So(cf.Count(), ShouldEqual, 98)
	})
}

/*
goos: darwin
goarch: arm64
pkg: github.com/wubingwei/notebook/filter/code
BenchmarkCuckooLookup-8   	 8726754	       136.0 ns/op	       7 B/op	       0 allocs/op
*/
func BenchmarkCuckooLookup(b *testing.B) {
	cf := cuckoo.NewFilter(1e8)
	var testObject int64 = 5e7
	for i := int64(1); i < testObject; i += 1 {
		cf.InsertUnique([]byte("wubingwei " + strconv.FormatInt(i, 10)))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cf.Lookup([]byte("wubingwei " + strconv.FormatInt(int64(i), 10)))
	}
}

type fingerprint byte
type bucket [bucketSize]fingerprint

const (
	bucketSize = 4
)

func getNextPow2(n uint64) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return uint(n)
}

func TestProductionCuckoo(t *testing.T) {
	Convey("CuckooProduction\n", t, func() {
		n := uint(1e7)

		cf := cuckoo.NewFilter(n)

		t.Logf("object number = %d", n)
		t.Logf("size of cuckooFilter = %d M-Bytes", unsafe.Sizeof(*cf)+unsafe.Sizeof(bucket{})*uintptr(getNextPow2(uint64(n))/bucketSize)>>20)

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
		var errCount int
		for scanner.Scan() {
			line := scanner.Text()
			num++
			if num >= 1e8 {
				break
			}
			if !cf.Insert([]byte(line)) {
				errCount++
			}
		}
		t.Logf("total object: %d", num)
		t.Logf("errCount object: %d", errCount)
		t.Logf("load elapsed = %f s", time.Since(start).Seconds())

		wd, _ := os.Getwd()
		t.Logf("wd: %s", wd)

		wFile, _ := os.Create("../_file/cuckoo_filter_binary_file")
		defer wFile.Close()
		t.Log("File created:", wFile.Name())

		w := bufio.NewWriter(wFile)
		bytesWritten := cf.Encode()
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("bytesWritten = %d M-Bytes", (unsafe.Sizeof(bytesWritten)+uintptr(len(bytesWritten)))>>20)
		w.Write(bytesWritten)

		t1 := time.Now()
		rFile := "../_file/cuckoo_filter_binary_file"
		bytesRead, err := os.ReadFile(rFile)
		if err != nil {
			log.Fatalf("Unable to read file: %s, error: %v", rFile, err)
		}

		cfCopy, err := cuckoo.Decode(bytesRead)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("cuckoo.Decode used %f Seconds", time.Since(t1).Seconds())

		// if !cmp.Equal(cf, cfCopy, cmp.AllowUnexported(cuckoo.Filter{})) {
		// 	t.Errorf("Decode = %v, want %v", cfCopy, cf)
		// }
		// t.Logf("cmp.Equal used %f Seconds", time.Since(t1).Seconds())

		start = time.Now()

		var errNumTest int64
		var testObject int64 = 100000
		for i := int64(1); i < testObject; i += 1 {
			if cfCopy.Lookup([]byte("wubingwei " + strconv.FormatInt(i, 10))) {
				errNumTest += 1
			}
		}
		t.Logf("Test Error Rate: %f, errNum = %d, testObject = %d\n, usedTime = %d us", float64(errNumTest)/float64(testObject), errNumTest, testObject, time.Since(start).Microseconds())

		t.Logf("Test wubingwei should be false, actual = %v", cfCopy.Lookup([]byte("wubingwei")))

		start = time.Now()
		runtime.GC()
		t.Logf("GC used %d ms", time.Since(start).Milliseconds())
	})
}
