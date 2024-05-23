package main

import (
	"errors"
	dio "github.com/dilfish/tools/io"
	"log"
	"sort"
	"strconv"
	"strings"
)

// BlockSize 使用 65536
// 此处理论上使用素数效率最高
// 为了方便调试，先使用 65536
// 65536 周围的素数是 65519 65521 65537 65539
// 这个值如果变小，将会节省内存，降低性能，退化为数组的二分查找
// 此处即先 65536-分查找，再二分查找
// 这个值如果变大，将会消耗内存，增加性能，减少二分查找范围
const BlockSize = 65536
const BlockNum = 65536

// Ranger 表示一个 IP 段，闭区间 [Start, End]
// NameIdx 指向 NameMap 的 key，值是 view str
type Ranger struct {
	Start   uint32
	End     uint32
	NameIdx uint32
}

// RangeBlock 表示同一个 C 段 IP
// 即，ip.uint32 >> 16 的值相等
type RangeBlock struct {
	// 有序数组
	List []Ranger
	Max  uint32
	Min  uint32
	Idx  int
	// 前一个非空的 block
	Prev *RangeBlock
}

type RangerMap struct {
	List [BlockSize]RangeBlock
	// key: 有序数字，value: view str
	ViewMap map[uint32]string
	// 反向 ViewMap
	RevViewMap map[string]uint32
	// 有序数当前值
	CurrIdx uint32
	SegNum  uint32
}

// Len 实现 Sort 接口
func (r RangeBlock) Len() int {
	return len(r.List)
}

// Less 实现 Sort 接口
func (r RangeBlock) Less(i, j int) bool {
	return r.List[i].Start < r.List[j].Start
}

// Swap 实现 Sort 接口
func (r RangeBlock) Swap(i, j int) {
	r.List[i], r.List[j] = r.List[j], r.List[i]
}

func NewRangerMap() *RangerMap {
	r := &RangerMap{
		ViewMap:    make(map[uint32]string),
		RevViewMap: make(map[string]uint32),
		CurrIdx:    1,
		// StatMap:    make(map[int]int),
	}
	for i := 0; i < BlockSize; i++ {
		r.List[i].Min = 0xFFFFFFFF
	}
	return r
}

// Manage 函数首先将 65536 个 block 中的分段排序
// 然后将每个 block 的 prev 都指向上一个非空的 block
func (r *RangerMap) Manage() {
	for i := 0; i < BlockSize; i++ {
		sort.Sort(r.List[i])
	}
	curr := 0
	// block 0 的 prev 指向 block 0 自己，即使 block 0 是空
	// 在遇到第一个非空 block 之前，所有的 block 的 prev 都是 block 0
	r.List[0].Prev = &r.List[0]
	for i := 1; i < BlockNum; i++ {
		r.List[i].Prev = &r.List[curr]
		if len(r.List[i].List) != 0 {
			curr = i
		}
		r.List[i].Idx = i
	}
	log.Println("ipv4 has segments:", r.SegNum)
	// for i := 0; i < BlockNum; i++ {
	// l := len(r.List[i].List)
	// if l > 1000 {
	// r.StatMap[i] = l
	// }
	// }
	// log.Println("range map:", len(r.List))
	// for k, v := range r.StatMap {
	// log.Println("block index:", k, ", block list len:", v, "min, max is:", r.List[k].Min, r.List[k].Max)
	// }
}

// getIndex 返回应该在的 block 的 index
func (r *RangerMap) getIndex(ip uint32) uint32 {
	return ip / BlockSize
}

// Add 添加一个 IP 段，start <= end
// 后续的 start 必须大于上一个 end
func (r *RangerMap) Add(start, end uint32, name string) {
	// log.Printf("add ip: 0x%x-0x%x: %s\n", start, end, name)
	r.SegNum = r.SegNum + 1
	nameIdx, ok := r.RevViewMap[name]
	if !ok {
		r.RevViewMap[name] = r.CurrIdx
		r.ViewMap[r.CurrIdx] = name
		nameIdx = r.CurrIdx
		r.CurrIdx = r.CurrIdx + 1
	}
	var ranger Ranger
	ranger.Start = start
	ranger.End = end
	ranger.NameIdx = nameIdx
	blockIndex := r.getIndex(start)
	r.List[blockIndex].List = append(r.List[blockIndex].List, ranger)
	if start < r.List[blockIndex].Min {
		r.List[blockIndex].Min = start
	}
	if end > r.List[blockIndex].Max {
		r.List[blockIndex].Max = end
	}
}

// Find 实现查找特定 IP 的 view str
func (r *RangerMap) Find(ip uint32) string {
	blockIndex := r.getIndex(ip)
	block := r.List[blockIndex]
	// 如果 block 为空，表示查找 IP 在之前的 block 中
	// 例如 ip 0x1111_0001 应该在 0x1111 block 中
	// 当 0x1111 block 为空时，表示上一个 block，即 block 0x1110
	// 的范围可能是 0x1110_0000 - 0x1111_ffff，一个 IP Range 占了两个 block
	// 本 block 的最小值小于待查找 IP 是同样的情况：上个 block 中的 IP Range 延伸到了本 block 中
	if len(block.List) == 0 || block.Min > ip {
		block = *r.List[blockIndex].Prev
	}
	// log.Println("block info:", blockIndex, block.Idx, block.Min, block.Max, block.List)

	// Search uses binary search to find and return the smallest index i
	// in [0, n) at which f(i) is true, assuming that on the range [0, n),
	// f(i) == true implies f(i+1) == true. That is, Search requires
	// that f is false for some (possibly empty) prefix of the input range [0, n) and
	// then true for the (possibly empty) remainder; Search returns the first true index.
	// If there is no such index, Search returns n. (Note that the "not found" return value is
	// not -1 as in, for instance, strings.Index.) Search calls f(i) only for i in the range [0, n).
	// Search 使用二分法查找升序数组
	// 返回值是最小的 true 值，即如果 lambda 函数返回 true，他会一直往前查找直到 false
	// 然后倒回去返回最小的那个 true 值
	// 当 End < ip 时，之前的 range 只会更小，不可能匹配，所以返回 false
	// 当 End == ip 时，应该返回 true，匹配到
	// 当 End > ip 时，ip 一定在本 range 或者更前面的 range 中
	// 由于 Search 使用最小的 true，所以返回的是 Start <= ip 的那个 range
	idx := sort.Search(len(block.List), func(i int) bool {
		// log.Println("find", i, block.List[i].Start, block.List[i].End)
		if block.List[i].End < ip {
			return false
		}
		return true
	})
	if idx < len(block.List) {
		viewIndex := block.List[idx].NameIdx
		return r.ViewMap[viewIndex]
	}
	log.Println("error as all default. this should not happen:", ip, blockIndex, idx, len(block.List))
	return "默认-默认-默认-默认-默认-默认"
}

func (r *RangerMap) cb(line string) error {
	array := strings.Split(line, " ")
	if len(array) != 3 {
		log.Println("array len is:", len(array), array)
		return errors.New("bad format")
	}
	start, err := strconv.ParseUint(array[0], 10, 32)
	if err != nil {
		log.Println("parse start error:", array[0], err)
		return err
	}
	end, err := strconv.ParseUint(array[1], 10, 32)
	if err != nil {
		log.Println("parse end error:", array[1], err)
		return err
	}
	view := array[2]
	r.Add(uint32(start), uint32(end), view)
	return nil
}

func (r *RangerMap) ReadFile(fn string) error {
	return dio.ReadLine(fn, r.cb)
}
