package bitcask_go

import "os"

type IndexType = int8

const (
	BTree IndexType = iota + 1 // Btree索引

	ART // Adaptive Radix Tree索引
)


type Options struct {
	DirPath string // 数据库数据目录

	DataFileSize int64 // 数据库文件大小

	SyncWrites bool // 每次写数据是否持久化

	IndexType IndexType // 索引类型
}

var DefaultOptions = Options{
	DirPath:      os.TempDir(),
	DataFileSize: 256 * 1024 * 1024, // 256MB
	SyncWrites:   false,
	IndexType:    BTree,
}


// IteratorOptions 索引迭代器配置项
type IteratorOptions struct {
	// 遍历前缀为指定值的 Key，默认为空
	Prefix []byte
	// 是否反向遍历，默认 false 是正向
	Reverse bool
}

var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}
