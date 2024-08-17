package index

import (
	"bitcask-go/data"
	"bytes"
	"github.com/google/btree"
)

// 抽象索引接口，目前接入google的btree，后续如果要更换其他数据结构，则可以直接实现这个接口
type Indexer interface {
	// Put 向索引中存储key对应的数据的位置信息
	Put(key []byte, pos *data.LogRecordPos) bool

	// Get 根据key取出对应的索引位置信息
	Get(key []byte) *data.LogRecordPos

	// Delete 根据key删除对应的索引位置信息
	Delete(key []byte) bool

	// Size 索引中的数据量
	Size() int

	// Iterator 迭代器
	Iterator(reverse bool) Iterator
}

type IndexType = int8

const (
	Btree IndexType = iota + 1 // Btree索引

	ART // Adaptive Radix Tree索引
)

func NewIndexer(typ IndexType) Indexer {
	switch typ {
	case Btree:
		return NewBTree()
	case ART:
		return nil
	default:
		panic("unsupported index type")
	}
}

type Item struct {
	key []byte
	pos *data.LogRecordPos
}

// 放入btree的item必须要实现这个Less方法，因为btree需要对item进行排序
func (ai *Item) Less(bi btree.Item) bool {
	return bytes.Compare(ai.key, bi.(*Item).key) == -1
}


// Iterator 通用索引迭代器
type Iterator interface {
	// Rewind 重新回到迭代器的起点，即第一个数据
	Rewind()

	// Seek 根据传入的 key 查找到第一个大于（或小于）等于的目标 key，根据从这个 key 开始遍历
	Seek(key []byte)

	// Next 跳转到下一个 key
	Next()

	// Valid 是否有效，即是否已经遍历完了所有的 key，用于退出遍历
	Valid() bool

	// Key 当前遍历位置的 Key 数据
	Key() []byte

	// Value 当前遍历位置的 Value 数据
	Value() *data.LogRecordPos

	// Close 关闭迭代器，释放相应资源
	Close()
}
