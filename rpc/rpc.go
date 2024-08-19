package rpc

import (
	bitcask "bitcask-go"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
)

type BitcaskService struct {
	db *bitcask.DB
}

func (b *BitcaskService) Put(kv map[string]string, reply *string) error {
	for key, value := range kv {
		if err := b.db.Put([]byte(key), []byte(value)); err != nil {
			return err
		}
	}
	*reply = "OK"
	return nil
}

func (b *BitcaskService) Get(key string, value *string) error {
	v, err := b.db.Get([]byte(key))
	if err != nil {
		return err
	}
	*value = string(v)
	return nil
}

func (b *BitcaskService) Delete(key string, reply *string) error {
	err := b.db.Delete([]byte(key))
	if err != nil {
		return err
	}
	*reply = "OK"
	return nil
}

func (b *BitcaskService) ListKeys(_ struct{}, keys *[]string) error {
	k := b.db.ListKeys()
	for _, key := range k {
		*keys = append(*keys, string(key))
	}
	return nil
}

func main() {
	// 初始化 DB 实例
	var err error
	options := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-rpc")
	options.DirPath = dir
	db, err := bitcask.Open(options)
	if err != nil {
		panic(fmt.Sprintf("failed to open db: %v", err))
	}
	defer db.Close()

	// 注册 RPC 服务
	bitcaskService := &BitcaskService{db: db}
	rpc.Register(bitcaskService)

	// 启动 RPC 服务
	listener, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}
	defer listener.Close()

	log.Printf("rpc server start at localhost:1234")
	rpc.Accept(listener)
}

