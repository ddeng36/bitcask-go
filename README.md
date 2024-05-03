# bitcask-go
利用go语言实现bitcask KV存储

## 内存和磁盘设计
### 内存
- 初版内存模型待用google的btree，后期加入adaptive radix tree和skip list。所有内存模型都需要实现Indexer接口
```go
type Indexer interface {
	// Put 向索引中存储key对应的数据的位置信息
	Put(key []byte, pos *data.LogRecordPos) bool

	// Get 根据key取出对应的索引位置信息
	Get(key []byte) *data.LogRecordPos

	// Delete 根据key删除对应的索引位置信息
	Delete(key []byte) bool
}
```

### 磁盘
- 初版磁盘采用标准系统文件IO，后期可以加入MMap内存映射。所有磁盘模型都需要实现IOManager接口
```go
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
```
- TODO 图

## 数据读写流程
### 写数据
- 先把数据写入磁盘，再更新内存索引,具体逻辑在db.Put()
- 删除也是写入一条log
```go
// Logecord 写入到数据文件的记录。日志：数据文件中的数据是追加写入的，类似日志格式
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType // 墓碑值，删除数据时需要使用
}
```
### 读数据
- 根据key去内存中查找。具体逻辑在db.Get()

- todo 图

## 数据库启动流程
1. 从磁盘中加载数据文件到内存中loadDataFiles()
2. 遍历文件中所有记录，并更新到内存索引中loadIndexFromDataFiles()