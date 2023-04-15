package compress

import (
	"github.com/pierrec/lz4/v4"
)

func Compress(fileBytes []byte, filesize int) ([]byte, error) { //wrapper for lz4 compression
	compressor := lz4.CompressorHC{Level: lz4.Level2} //TODO: might move lvl to config later on
	size := lz4.CompressBlockBound(filesize)
	compressedBuffer := make([]byte, size)
	compressedSize, err := compressor.CompressBlock(fileBytes, compressedBuffer)
	if err != nil {
		return nil, err
	}
	return compressedBuffer[:compressedSize], nil
}
