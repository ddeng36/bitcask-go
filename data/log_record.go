package data

// 数据的内存索引，主要描述数据在磁盘上的位置
type LogRecordPos struct {
	Fid    uint32 // 文件id，表示数据存储到到了哪个文件当中
	Offset int64  // 偏移量，表示数据村书到了文件当中的位置
}
