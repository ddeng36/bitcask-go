package bitcask_go

import (
	"bitcask-go/data"
	"bitcask-go/index"
	"errors"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// DB bitcask 存储引擎实例
type DB struct {
	options    Options
	mu         *sync.RWMutex
	filesIds   []int                     // 文件id，只能在加载索引的时候用
	activeFile *data.DataFile            // 当前活跃数据文件，可以用于写入
	olderFiles map[uint32]*data.DataFile // 就的数据文件，只能读
	index      index.Indexer             // 内存索引
}

// Open 打开bitcask存储引擎实例
func Open(options Options) (*DB, error) {
	// 校验
	if err := checkOptions(options); err != nil {
		return nil, err
	}
	if _, err := os.Stat(options.DirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(options.DirPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db := &DB {
		options:	options,
		mu:			&sync.RWMutex{},
		olderFiles:	make(map[uint32]*data.DataFile),
		index: 		index.NewIndexer(options.IndexType),
	}

	// 从磁盘加载数据文件到内存
	if err := db.loadDataFiles(); err != nil {
		return nil, err
	}

	// 遍历文件中所有记录，并更新到内存索引中
	if err := db.loadIndexFromDataFiles(); err != nil {
		return nil, err
	}

	return db, nil
}

// Put 写入KV数据
func (db *DB) Put(key []byte, value []byte) error {
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}

	// 追加写如到当前活跃数据文件中
	logRecord := &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordNormal,
	}
	pos, err := db.appendLogRecord(logRecord)
	if err != nil {
		return err
	}

	// 更新内存索引
	if ok := db.index.Put(key, pos); !ok {
		return ErrIndexUpdateFailed
	}
	return nil
}

// Get 根据k拿到v
func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if len(key) == 0 {
		return nil, ErrKeyIsEmpty
	}

	// 从内存中获取key的索引信息，如果内存中无所引则key不存在
	logRecordPos := db.index.Get(key)
	if logRecordPos == nil {
		return nil, ErrKeyNotFound
	}

	// 根据FID找到对应的数据文件
	var dataFile *data.DataFile
	if db.activeFile.FileId == logRecordPos.Fid {
		dataFile = db.activeFile
	} else {
		dataFile = db.olderFiles[logRecordPos.Fid]
	}
	if dataFile == nil {
		return nil, ErrDataFileNotFound
	}

	// 根据偏移量读取数据
	logRecord, _, err := dataFile.ReadLogRecord(logRecordPos.Offset)
	if err != nil {
		return nil, err
	}
	if logRecord.Type == data.LogRecordDeleted {
		return nil, ErrKeyNotFound
	}
	return logRecord.Value, nil
}

// 追加写数据到活跃文件中，删除也用的这个
func (db *DB) appendLogRecord(logRecord *data.LogRecord) (*data.LogRecordPos, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 判断当前活跃数据文件是否存在，如国为空则初始化一个
	if db.activeFile == nil {
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}

	// 写入数据编码
	encRecord, size := data.EncodeLogRecord(logRecord)

	// 如果写入的数据已经到达活跃文件的阈值，则关闭活跃文件并打开心的文件
	if db.activeFile.WriteOff+size > db.options.DataFileSize {
		// 持久化数据文件
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}
		// 将当前的活跃文件转化为旧的数据文件
		// 注意，这里复制的是引用，go是值传递
		db.olderFiles[db.activeFile.FileId] = db.activeFile

		// 打开新的数据文件
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}

	writeOff := db.activeFile.WriteOff
	if err := db.activeFile.Write(encRecord); err != nil {
		return nil, err
	}

	// 根据用户配置决定是否持久化
	if db.options.SyncWrites {
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}
	}
	// 构造内存索引信息
	pos := &data.LogRecordPos{
		Fid:    db.activeFile.FileId,
		Offset: writeOff,
	}
	return pos, nil
}

// 设置当前活跃文件。该方为必须持有互斥锁
func (db *DB) setActiveDataFile() error {
	// 获取id，默认为0或db的activefile的下一位
	var initialFileId uint32 = 0
	if db.activeFile != nil {
		initialFileId = db.activeFile.FileId + 1
	}

	// 根据id打开新的数据文件
	dataFile, err := data.OpenDataFile(db.options.DirPath, initialFileId)
	if err != nil {
		return err
	}
	db.activeFile = dataFile
	return nil

}

// 从磁盘中加载数据文件到内存中
func (db *DB) loadDataFiles() error {
	// 找到目录
	dirEntries, err := os.ReadDir(db.options.DirPath)
	if err != nil {
		return err
	}

	// 遍历目录并拿到“.data”文件名结尾的文件
	var fileIds []int
	for _, entry := range dirEntries {
		if strings.HasSuffix(entry.Name(), data.DataFileNameSuffix) {
			splitNames := strings.Split(entry.Name(), ".")
			fileId, err := strconv.Atoi(splitNames[0])
			if err != nil {
				return ErrDataDirectoryCorrupted
			}
			fileIds = append(fileIds, fileId)
		}
	}

	// 对文件id排序，从小到大加载
	sort.Ints(fileIds)
	db.filesIds = fileIds

	// 遍历每个文件id，打开对应的数据文件
	for i, fid := range fileIds {
		dataFile, err := data.OpenDataFile(db.options.DirPath, uint32(fid))
		if err != nil {
			return err
		}
		if i == len(fileIds)-1 {
			db.activeFile = dataFile
		} else {
			db.olderFiles[uint32(fid)] = dataFile
		}
	}
	return nil
}

// 遍历文件中所有记录，并更新到内存索引中
func (db *DB) loadIndexFromDataFiles() error {
	if len(db.filesIds) == 0 {
		return nil
	}

	for i, fid := range db.filesIds {
		var fileId = uint32(fid)
		var dataFile *data.DataFile
		if fileId == db.activeFile.FileId {
			dataFile = db.activeFile
		} else {
			dataFile = db.olderFiles[fileId]
		}

		var offset int64 = 0
		for {
			logRecord, size, err := dataFile.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			//构造内存中的索引并保存
			logRecordPos := &data.LogRecordPos{
				Fid:    fileId,
				Offset: offset,
			}
			if logRecord.Type == data.LogRecordDeleted {
				db.index.Delete(logRecord.Key)
			} else {
				db.index.Put(logRecord.Key, logRecordPos)
			}

			// 递增offset，下一次从新的位置开始
			offset += size
		}

		// 如果是活跃文件，更新这个文件的 WriteOdd
		if i == len (db.filesIds) - 1 {
			db.activeFile.WriteOff = offset
		}
	}
	return nil
}

func checkOptions(options Options) error {
	if options.DirPath == "" {
		return errors.New("the database directory is empty")
	}
	if options.DataFileSize == 0 {
		return errors.New("the data file size is 0")
	}
	return nil
}
