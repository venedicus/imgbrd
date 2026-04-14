package util

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func IsAllowedImage(header *multipart.FileHeader) bool {
	contentType := header.Header.Get("Content-Type")
	return allowedImageTypes[contentType]
}

func SaveUploadedFile(file multipart.File, header *multipart.FileHeader, dir string, filename string) (string, error) {
	if header.Size == 0 {
		return "", nil
	}

	if !IsAllowedImage(header) {
		return "", errors.New("invalid file type")
	}

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", err
	}

	savePath := filepath.Join(dir, filename)

	dst, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return savePath, nil
}