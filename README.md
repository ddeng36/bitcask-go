# bitcask-go
利用go语言实现bitcask KV存储

## 内存和磁盘设计
- 初版内存模型待用google的btree，后期加入adaptive radix tree和skip list。所有内存模型都需要实现Indexer接口
- 初版磁盘采用标准系统文件IO，后期可以加入MMap内存映射。所有磁盘模型都需要实现IOManager接口
- TODO 图