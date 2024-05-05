package bitcask_go

import "os"

type Options struct {
	DirPath string // 数据库数据目录

	DataFileSize int64 // 数据库文件大小

	SyncWrites bool // 每次写数据是否持久化

	IndexType IndexType // 索引类型
}
type IndexType = int8

const (
	BTree IndexType = iota + 1 // Btree索引

	ART // Adaptive Radix Tree索引
)


var DefaultOptions = Options{
	DirPath:      os.TempDir(),
	DataFileSize: 256 * 1024 * 1024, // 256MB
	SyncWrites:   false,
	IndexType:    BTree,
}