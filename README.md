# bitcask-go
利用go语言实现bitcask KV存储。<a src=".\resources\bitcask-intro.pdf">论文详情</a>
## Bitcask架构
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


## 特性
### 事务
#### Seq编码Log
- 在DB中维护一个全局递增的seq用于encode Log，从而支持事务
<img src=".\resources\transaction_log.png">bitcast读数据的过程</img>

#### 事务的Put,Delete,Commit
- 执行逻辑
  1. Put和Delete方法不会更新内存或磁盘，而是更新map，，map中存储的是最后一次更新的结果
  2. Commit会提交变动，先得到全局加锁递增的seq num，然后把seqnum与key进行编码。之后，先更新磁盘，再写入Fin日志，最后更新内存。
<img src=".\resources\write_batch_commit.png">事务的Put,Delete,Commit过程</img>
#### 事务的Load
  1. Load时，先从硬盘load进Datafile中，然后判断是否为事务操作，如果否则直接载入内存索引，如果是则暂存进map中。遍历map，判断事务是否有Fin标识，如果无则不载入内存，从用户角度看，保证了事务的原子性。
  <img src=".\resources\write_batch_load.png">事务的Load过程</img>

 ### 冗余数据清理 - Merge
- 逻辑
  - merge用来清理磁盘上的无用数据：transaction seq num，deleted log。新进程调用Merge时，会遍历datafile目录，并把合法的log写入Mergeddatafile和hintfile
<img src=".\resources\merge.png">merge过程</img>

  - 当load时，会先检查是否有Merge文件，如果有，则把MergedFileData载入Datafile中，把hintfile载入index中。
<img src=".\resources\merge.png">merge的load过程</img>

### 内存索引优化
#### 支持多种数据结构索引
	- B树：key+索引维护在内存中，val维护在磁盘中。查找数据需要经历1次I/O。由于key+索引维护在内存中，所以数据量受限与内存空间。
	- ART：key+索引维护在内存中，val维护在磁盘中。查找数据需要经历1次I/O。key+索引维护在内存中，val维护在磁盘中。查找数据需要经历1次I/O。
	- B+数：key+索引，val都维护在磁盘中。查找数据需要经过2次I/O，不受限与内存空间，但读写性能下降。
<img src=".\resources\data_indexer.png">indexer implements</img>

#### 索引的锁的优化
	- 之前内存中维护了一个索引结构，所有的读写操作都会竞争这个索引的锁，在高并发的场景下可能是一个性能瓶颈。我们可以维护所个索引，通过hash函数取模映射到不同的索引中。这样竞争锁的概率下降了。
	- 如果存在多个索引结构，则迭代器不可用了，为了解决这个问题，引入了最小堆。
<img src=".\resources\multi_indexer.png">multi_indexer</img>
<img src=".\resources\min_heap.png">min_heap</img>

### 文件I/O优化
#### 文件锁
- bitcask为单机存储，只允许在但个进程中运行，加入flock为文件上所，实现锁进程互斥。
#### 个性化持久化
- SyncWrite选项用于控制是否每次写入都持久化到磁盘，false为系统自动调度，true为每次写入。
- 加入BytesPerSync，累积到一定字节数，进行持久化。
#### 启动速度优化
- 之前，启动时需要将磁盘中的数据加载到内存中。在这中默认的文件I/O下，需要由操作系统将数据从内核态拷贝到用户态
- 优化后采用内存文件映射（MMAP）IO来加速启动速度，避免了从内核态拷贝到用户态

### 数据备份
Copy()可以将数据拷贝到指定位置，用于支持数据备份

### 支持HTTP和RPC
实现了HTTP接口和RPC接口，外部可以通过网络或者远程调用bitcask

