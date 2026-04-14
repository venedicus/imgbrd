package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SiteTitle     string
	PublicAddr    string
	AdminAddr     string
	AdminToken    string
	DefaultTheme  string
	MaxUploadMB   int
	WebhookURL    string
	WebhookSecret string
	EnablePprof   bool
	PprofAddr     string
}

func Load() Config {
	c := Config{
		SiteTitle:     getenv("IMGBRD_SITE_TITLE", "imgbrd"),
		PublicAddr:    getenv("IMGBRD_PUBLIC_ADDR", ":8080"),
		AdminAddr:     strings.TrimSpace(os.Getenv("IMGBRD_ADMIN_ADDR")),
		AdminToken:    strings.TrimSpace(os.Getenv("IMGBRD_ADMIN_TOKEN")),
		DefaultTheme:  getenv("IMGBRD_DEFAULT_THEME", "futaba"),
		MaxUploadMB:   25,
		WebhookURL:    strings.TrimSpace(os.Getenv("IMGBRD_WEBHOOK_URL")),
		WebhookSecret: strings.TrimSpace(os.Getenv("IMGBRD_WEBHOOK_SECRET")),
		EnablePprof: strings.EqualFold(strings.TrimSpace(os.Getenv("IMGBRD_ENABLE_PPROF")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("IMGBRD_ENABLE_PPROF")), "true"),
		PprofAddr:   strings.TrimSpace(os.Getenv("IMGBRD_PPROF_ADDR")),
	}
	if c.EnablePprof && c.PprofAddr == "" {
		c.PprofAddr = "127.0.0.1:6060"
	}
	if v := strings.TrimSpace(os.Getenv("IMGBRD_MAX_UPLOAD_MB")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.MaxUploadMB = n
		}
	}
	return c
}

func (c *Config) MaxUploadBytes() int64 {
	if c.MaxUploadMB <= 0 {
		return 25 << 20
	}
	return int64(c.MaxUploadMB) << 20
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
