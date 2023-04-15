package files

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

func ValidateImage(fileBytes []byte, fileType string) bool {
	var imageFile image.Image
	var err error
	logger.Debug.Println(fileType)
	r := bytes.NewReader(fileBytes)
	switch fileType {
	case ".jpg":
		fallthrough
	case ".jpeg":
		if imageFile, err = jpeg.Decode(r); err != nil {
			return false
		}
	case ".png":
		if imageFile, err = png.Decode(r); err != nil {
			return false
		}
	case ".gif":
		if imageFile, err = gif.Decode(r); err != nil {
			return false
		}
	default:
		return false
	}
	bounds := imageFile.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > config.Config.Server.ImageProfileSize || height > config.Config.Server.ImageProfileSize {
		logger.Debug.Println("bad 1")
		return false
	}
	logger.Debug.Println(width == height)
	return width == height
}
