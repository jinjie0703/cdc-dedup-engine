package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// GearTable 预生成的哈希表，用于快速计算字节流滚动哈希
var GearTable = [256]uint32{
	0x6735518b, 0x5821bb48, 0x18e8dfac, 0x48ed3383, 0x0e5d95e2, 0x98bb77a1, 0x7685a7bb, 0x05f0dfab,
	0x2c6a0c0b, 0x1f0611e9, 0x56a6ec15, 0x5a18d18e, 0x51c720e3, 0x403d5272, 0x76426477, 0x367f0b83,
	0x4037bc86, 0x2213cfca, 0x0852e648, 0x6e25777a, 0x53460e53, 0x633e7221, 0x47e8fc9f, 0x4efae018,
	0x0283c7dc, 0x3d0bbf12, 0x334bb013, 0x0b8a4fcf, 0x66be75ee, 0x13c3b0dd, 0x0eb34f1d, 0x187ffca9,
	0x391f6ae2, 0x4d63fa26, 0x41e13cb3, 0x6e52292f, 0x4386fc75, 0x3e185f26, 0x261ef4a8, 0x2e061806,
	0x546c1ae3, 0x272445e9, 0x0e118086, 0x476bfe46, 0x1f3c3acb, 0x51ef0a64, 0x64faee02, 0x43391d1e,
	0x5e08670b, 0x368bb03b, 0x06bd8ef7, 0x0b81561f, 0x4e65ecab, 0x0c93abdf, 0x2a3e5cbb, 0x18db7ff2,
	0x651c6e17, 0x58d601d5, 0x1a87b1c3, 0x2eb962f3, 0x05e19ec3, 0x4f174775, 0x6a1005f0, 0x0a91f5e8,
	0x6b63fa26, 0x46ef3286, 0x6c6e9be1, 0x2a4413e1, 0x0d5c3f3d, 0x3f62e841, 0x49e6cc86, 0x3e10fa65,
	0x1d467f58, 0x431b9efb, 0x1776bb53, 0x4376fb1b, 0x0e964d85, 0x69ed2841, 0x133d1b82, 0x5e23cb23,
	0x3e21cb32, 0x4f16a664, 0x5381fbb4, 0x66088d1d, 0x3f46f345, 0x629f6048, 0x26955e82, 0x696b9f71,
	0x153e4bcf, 0x5824c6e8, 0x3d31fe13, 0x0e3a6c08, 0x6c6e4e83, 0x3c788647, 0x41f3f4ab, 0x1e3678ca,
	0x5b38ed31, 0x2863a436, 0x62391ce4, 0x0b81fa39, 0x4e83efbb, 0x1b2c4d63, 0x1a89f315, 0x633d7124,
	0x1634fe73, 0x347c6e64, 0x49c6d36e, 0x0e88fe7b, 0x3101eb65, 0x583e74be, 0x463df4bc, 0x2e061806,
	0x1378ee0c, 0x092b77a1, 0x58448ea4, 0x4a18bcf3, 0x2a3e5cbb, 0x66c77a34, 0x384bb013, 0x62a8ef32,
	0x391f6ae2, 0x4d63fa26, 0x41e13cb3, 0x6e52292f, 0x4386fc75, 0x3e185f26, 0x261ef4a8, 0x2e061806,
	0x546c1ae3, 0x272445e9, 0x0e118086, 0x476bfe46, 0x1f3c3acb, 0x51ef0a64, 0x64faee02, 0x43391d1e,
	0x5e08670b, 0x368bb03b, 0x06bd8ef7, 0x0b81561f, 0x4e65ecab, 0x0c93abdf, 0x2a3e5cbb, 0x18db7ff2,
	0x651c6e17, 0x58d601d5, 0x1a87b1c3, 0x2eb962f3, 0x05e19ec3, 0x4f174775, 0x6a1005f0, 0x0a91f5e8,
	0x6b63fa26, 0x46ef3286, 0x6c6e9be1, 0x2a4413e1, 0x0d5c3f3d, 0x3f62e841, 0x49e6cc86, 0x3e10fa65,
	0x1d467f58, 0x431b9efb, 0x1776bb53, 0x4376fb1b, 0x0e964d85, 0x69ed2841, 0x133d1b82, 0x5e23cb23,
	0x3e21cb32, 0x4f16a664, 0x5381fbb4, 0x66088d1d, 0x3f46f345, 0x629f6048, 0x26955e82, 0x696b9f71,
	0x153e4bcf, 0x5824c6e8, 0x3d31fe13, 0x0e3a6c08, 0x6c6e4e83, 0x3c788647, 0x41f3f4ab, 0x1e3678ca,
	0x5b38ed31, 0x2863a436, 0x62391ce4, 0x0b81fa39, 0x4e83efbb, 0x1b2c4d63, 0x1a89f315, 0x633d7124,
	0x1634fe73, 0x347c6e64, 0x49c6d36e, 0x0e88fe7b, 0x3101eb65, 0x583e74be, 0x463df4bc, 0x2e061806,
	0x1378ee0c, 0x092b77a1, 0x58448ea4, 0x4a18bcf3, 0x2a3e5cbb, 0x66c77a34, 0x384bb013, 0x62a8ef32,
	0x391f6ae2, 0x4d63fa26, 0x41e13cb3, 0x6e52292f, 0x4386fc75, 0x3e185f26, 0x261ef4a8, 0x2e061806,
	0x546c1ae3, 0x272445e9, 0x0e118086, 0x476bfe46, 0x1f3c3acb, 0x51ef0a64, 0x64faee02, 0x43391d1e,
	0x5e08670b, 0x368bb03b, 0x06bd8ef7, 0x0b81561f, 0x4e65ecab, 0x0c93abdf, 0x2a3e5cbb, 0x18db7ff2,
	0x651c6e17, 0x58d601d5, 0x1a87b1c3, 0x2eb962f3, 0x05e19ec3, 0x4f174775, 0x6a1005f0, 0x0a91f5e8,
	0x6b63fa26, 0x46ef3286, 0x6c6e9be1, 0x2a4413e1, 0x0d5c3f3d, 0x3f62e841, 0x49e6cc86, 0x3e10fa65,
	0x1d467f58, 0x431b9efb, 0x1776bb53, 0x4376fb1b, 0x0e964d85, 0x69ed2841, 0x133d1b82, 0x5e23cb23,
}

// Chunk 代表切分后的单个数据块元数据
type Chunk struct {
	Data []byte
	Hash string
	Size int
}

// CDCChunker FastCDC 切块器结构
type CDCChunker struct {
	MinSize int
	AvgSize int
	MaxSize int
	Mask    uint32
}

// NewCDCChunker 创建一个新切块器
func NewCDCChunker(minSize, avgSize, maxSize int) *CDCChunker {
	return &CDCChunker{
		MinSize: minSize,
		AvgSize: avgSize,
		MaxSize: maxSize,
		Mask:    0x7FFF, // 15-bit zero mask (~32KB-64KB avg)
	}
}

// ChunkStream 读取 reader 并在流中动态切块
func (c *CDCChunker) ChunkStream(r io.Reader, onChunk func(Chunk) error) error {
	bufSize := 2 * 1024 * 1024 // 2MB 读取缓冲
	buffer := make([]byte, 0, bufSize)
	temp := make([]byte, bufSize)

	for {
		n, err := r.Read(temp)
		if n > 0 {
			buffer = append(buffer, temp[:n]...)
		}

		// 当缓冲里的数据超过最大块大小时，不断切割
		for len(buffer) >= c.MaxSize {
			cut := c.findBoundary(buffer)
			chunkData := make([]byte, cut)
			copy(chunkData, buffer[:cut])
			buffer = buffer[cut:]

			hash := sha256.Sum256(chunkData)
			hashStr := hex.EncodeToString(hash[:])

			if err := onChunk(Chunk{
				Data: chunkData,
				Hash: hashStr,
				Size: cut,
			}); err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// 处理末尾剩余的数据
	for len(buffer) > 0 {
		cut := len(buffer)
		if len(buffer) > c.MaxSize {
			cut = c.findBoundary(buffer)
		}

		chunkData := make([]byte, cut)
		copy(chunkData, buffer[:cut])
		buffer = buffer[cut:]

		hash := sha256.Sum256(chunkData)
		hashStr := hex.EncodeToString(hash[:])

		if err := onChunk(Chunk{
			Data: chunkData,
			Hash: hashStr,
			Size: cut,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (c *CDCChunker) findBoundary(data []byte) int {
	n := len(data)
	if n <= c.MinSize {
		return n
	}

	limit := n
	if limit > c.MaxSize {
		limit = c.MaxSize
	}

	var hashVal uint32
	for i := c.MinSize; i < limit; i++ {
		hashVal = (hashVal << 1) + GearTable[data[i]]
		if (hashVal & c.Mask) == 0 {
			return i + 1
		}
	}

	return limit
}
