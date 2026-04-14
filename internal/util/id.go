package util

import (
	"fmt"
	"time"
)

func GenerateFileName(original string) string {
	ext := filepathExt(original)
	return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
}

func filepathExt(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
	}
	return ""
}