package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Backend 抽象存储后端接口
type Backend interface {
	Put(hash string, data []byte) error
	Get(hash string) ([]byte, error)
	Delete(hash string) error
	Exists(hash string) (bool, error)
}

// LocalStorage 本地磁盘 CAS 存储后端
type LocalStorage struct {
	RootDir string
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(rootDir string) *LocalStorage {
	return &LocalStorage{RootDir: rootDir}
}

// hashPath 生成两级目录路径例如: root/ab/cd/abcd1234...
func (l *LocalStorage) hashPath(hash string) string {
	if len(hash) < 4 {
		return filepath.Join(l.RootDir, hash)
	}
	return filepath.Join(l.RootDir, hash[:2], hash[2:4], hash)
}

func (l *LocalStorage) Put(hash string, data []byte) error {
	path := l.hashPath(hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	// 如果已经存在，说明是完全相同的块，无需再次写入物理磁盘
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, data, 0644)
}

func (l *LocalStorage) Get(hash string) ([]byte, error) {
	path := l.hashPath(hash)
	return os.ReadFile(path)
}

func (l *LocalStorage) Delete(hash string) error {
	path := l.hashPath(hash)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *LocalStorage) Exists(hash string) (bool, error) {
	path := l.hashPath(hash)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CloudStorage 预留远端云端对象存储实现 (兼容 COS / OSS / S3)
type CloudStorage struct {
	Endpoint  string
	Bucket    string
	SecretID  string
	SecretKey string
}

func NewCloudStorage(endpoint, bucket, id, key string) *CloudStorage {
	return &CloudStorage{Endpoint: endpoint, Bucket: bucket, SecretID: id, SecretKey: key}
}

func (c *CloudStorage) Put(hash string, data []byte) error {
	// TODO: 调用腾讯云 COS / 阿里云 OSS SDK 上传 object
	return fmt.Errorf("cloud storage not initialized")
}

func (c *CloudStorage) Get(hash string) ([]byte, error) {
	return nil, fmt.Errorf("cloud storage not initialized")
}

func (c *CloudStorage) Delete(hash string) error {
	return fmt.Errorf("cloud storage not initialized")
}

func (c *CloudStorage) Exists(hash string) (bool, error) {
	return false, nil
}
