package fio
const DataFilePerm = 0644

type IOManager interface {
	// Read 从文件给定的位置开始读取数据
	Read([]byte, int64) (int, error)

	// Write 写入字节数组到文件中
	Write([]byte) (int, error)

	// Sync 持久化数据
	Sync() error

	// Close 关闭文件
	Close() error
}