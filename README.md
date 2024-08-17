# bitcask-go
利用go语言实现bitcask KV存储。<a src=".\resources\bitcask-intro.pdf">论文详情</a>
## Bitcask
### DB Server
- 一个Bitcask实例是一个目录。同一时间内OS只能有一个进程（可以理解成DB server）对Bitcask进行写操作。
- bitcask由磁盘和内存两部分组成，activeFile和olderFiles属于磁盘部分，index属于内存索引
```go
	// DB bitcask 存储引擎实例
	type DB struct {
		options    Options
		mu         *sync.RWMutex
		filesIds   []int                     // 文件id，只能在加载索引的时候用
		activeFile *data.DataFile            // 当前活跃数据文件，可以用于写入
		olderFiles map[uint32]*data.DataFile // 就的数据文件，只能读
		index      index.Indexer             // 内存索引
	}
```
bitcask方法接口
```go
	Open(Directory Name)
	//打开一个 bitcask 数据库实例，使用传入的目录路径
	// 需要保证进程对该目录具有可读可写权限

	Get(Key)
	// 通过 Key 获取存储的 value

	Put(Key,Value)
	//存储 key 和 value

	Delete(Key)
	// 删除一个 key

	Listkeys()
	//获取全部的 key

	Fold(Fun)
	// 遍历所有的数据，执行函数 Fun

	Merge(Directory Name)
	//执行 merge，清理无效数据

	Sync()
	//刷盘，将所有内核缓冲区的写入持久化到磁盘中

	Close()
	// 关闭数据库
```
### 磁盘
#### IO管理
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
#### bitcask的磁盘
- 在任何时间，只有一个data file是active的，当file被写满时，该file会被close，并成为old data file，old data file永远不会被再次写。
- file采用了顺序追加的格式，减少了磁盘寻址的时间。
<img src=".\resources\bitcask_on_disk.png"/>



#### entry / log_record
- bitcask的写入过程就是追加一条entry到active data file的最后。
- 删除也是一次追加，但是要写入不同的墓碑值。每当merge时，会清空这些无效数据
	<img src=".\resources\entry.png">单条entry</img>
	```go
	// entry的数据结构
	// LogRecord 的头部信息
	type logRecordHeader struct {
		crc        uint32        // crc 校验值
		recordType LogRecordType // 标识 LogRecord 的类型
		keySize    uint32        // key 的长度
		valueSize  uint32        // value 的长度
	}
	// Logecord 写入到数据文件的记录。日志：数据文件中的数据是追加写入的，类似日志格式。
	type LogRecord struct {
		Key   []byte
		Value []byte
		Type  LogRecordType // tombstone墓碑值，删除数据时需要使用
	}
	```
	<img src="resources\active_file.png">active data file 就是多条entry的合集</img>


### 内存
#### 索引
- bitcask的内存中的索引可以采用HashMap，Btree,B+ tree。初版内存模型待用google的btree，后期加入adaptive radix tree和skip list。所有内存模型都需要实现Indexer接口
	```go
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
	```
#### 索引节点
	```go
	// 数据的内存索引，主要描述数据在磁盘上的位置
	type LogRecordPos struct {
		Fid    uint32 // 文件id，表示数据存储到到了哪个文件当中
		Offset int64  // 偏移量，表示数据村书到了文件当中的位置
	}
	```
	<img src=".\resources\bitcask_on_memory.png">bitcast的内存模型</img>
#### 索引迭代器
	```go
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
	```

### 数据读写流程
#### 写数据
- 先把数据写入磁盘，再更新内存索引,具体逻辑在db.Put()
<img src=".\resources\put.png">bitcast读数据的过程</img>

#### 读数据
- 根据key去内存中查找。具体逻辑在db.Get()
<img src=".\resources\get.png">bitcast读数据的过程</img>

#### 加载数据
1. loadDataFiles()从磁盘中加载数据文件到内存中，把DataFile放在DB的actvieFile和olderFiles中
2. loadIndexFromDataFiles()遍历DB的actvieFile和olderFiles，通过offset读出LogRecord，并通过db.index.Put(logRecord.Key, logRecordPos)更新到内存索引中。
<img src=".\resources\load.png">bitcast读数据的过程</img>

