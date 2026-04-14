package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

type mediaResult struct {
	URL   string
	Thumb string
	Hash  string
	Mime  string
	Size  int64
}

func (h *Handler) processMediaField(r *http.Request, field string) (*mediaResult, error) {
	fh, header, err := r.FormFile(field)
	if err != nil {
		return nil, nil
	}
	defer fh.Close()

	if header.Size == 0 {
		return nil, nil
	}

	max := h.cfg.MaxUploadBytes()
	if header.Size > max {
		return nil, fmt.Errorf("файл больше %d МБ", h.cfg.MaxUploadMB)
	}

	data, err := io.ReadAll(io.LimitReader(fh, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("файл больше %d МБ", h.cfg.MaxUploadMB)
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	if img, th, ok := h.svc.FindDedupMedia(hash); ok && img != "" {
		return &mediaResult{URL: img, Thumb: th, Hash: hash, Mime: "", Size: int64(len(data))}, nil
	}

	mime := sniffMedia(data)
	if mime == "" {
		return nil, errors.New("неподдерживаемый тип файла")
	}

	ext := extForMime(mime)
	baseName := hash[:24] + ext
	savePath := filepath.Join("static", "uploads", baseName)
	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(savePath, data, 0o644); err != nil {
		return nil, err
	}
	pub := "/static/uploads/" + baseName

	var thumbPath string
	if strings.HasPrefix(mime, "image/") {
		if thFile, err := buildThumb(savePath, hash[:20]); err == nil && thFile != "" {
			thumbPath = "/static/uploads/" + filepath.Base(thFile)
		}
	}

	return &mediaResult{
		URL:   pub,
		Thumb: thumbPath,
		Hash:  hash,
		Mime:  mime,
		Size:  int64(len(data)),
	}, nil
}

func buildThumb(srcPath, idPrefix string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}
	w := 200
	if img.Bounds().Dx() <= w {
		w = img.Bounds().Dx()
	}
	thumb := imaging.Resize(img, w, 0, imaging.Lanczos)
	out := filepath.Join("static", "uploads", "t_"+idPrefix+".jpg")
	of, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer of.Close()
	if err := jpeg.Encode(of, thumb, &jpeg.Options{Quality: 82}); err != nil {
		return "", err
	}
	return out, nil
}

func sniffMedia(b []byte) string {
	if len(b) < 12 {
		return ""
	}
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "image/jpeg"
	case len(b) >= 8 && b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G':
		return "image/png"
	case len(b) >= 6 && b[0] == 'G' && b[1] == 'I' && b[2] == 'F':
		return "image/gif"
	case len(b) >= 12 && b[0] == 'R' && b[1] == 'I' && b[2] == 'F' && b[3] == 'F':
		if bytes.Contains(b[:min(32, len(b))], []byte("WEBP")) {
			return "image/webp"
		}
	case len(b) >= 4 && b[0] == 0x1A && b[1] == 0x45 && b[2] == 0xDF && b[3] == 0xA3:
		return "video/webm"
	case len(b) >= 12 && b[4] == 'f' && b[5] == 't' && b[6] == 'y' && b[7] == 'p':
		return "video/mp4"
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extForMime(m string) string {
	switch m {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/webm":
		return ".webm"
	case "video/mp4":
		return ".mp4"
	default:
		return ".bin"
	}
}

// legacy wrapper for old isImage-only path
func (h *Handler) handleUpload(r *http.Request) (string, error) {
	m, err := h.processMediaField(r, "image")
	if err != nil || m == nil {
		return "", err
	}
	return m.URL, nil
}
