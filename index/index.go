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
}

type Item struct {
	key []byte
	pos *data.LogRecordPos
}

func (ai *Item) Less(bi btree.Item) bool {
	return bytes.Compare(ai.key, bi.(*Item).key) == -1
}
