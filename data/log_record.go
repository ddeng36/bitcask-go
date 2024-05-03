package data

type LogRecordType = byte

const (
	LogRecordNormal LogRecordType = iota
	LogRecordDeleted
)

// Logecord 写入到数据文件的记录。日志：数据文件中的数据是追加写入的，类似日志格式
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType // 墓碑值，删除数据时需要使用
}

// 数据的内存索引，主要描述数据在磁盘上的位置
type LogRecordPos struct {
	Fid    uint32 // 文件id，表示数据存储到到了哪个文件当中
	Offset int64  // 偏移量，表示数据村书到了文件当中的位置
}

// EnCodeLogRecord 对 LogRecord 进行编码，返回字节数组及其长度
func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {
	return nil, 0
}
