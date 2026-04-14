package theme

import (
	"net/http"
	"strings"
)

const CookieName = "imgbrd_theme"

// IDs must match static/css/themes/{id}.css
var Valid = map[string]struct{}{
	"futaba":   {},
	"tomorrow": {},
	"neutral":  {},
	"win95":    {},
}

func IsValid(id string) bool {
	_, ok := Valid[id]
	return ok
}

// Resolve returns the active theme: cookie if valid, else default.
func Resolve(r *http.Request, defaultID string) string {
	if c, err := r.Cookie(CookieName); err == nil && IsValid(c.Value) {
		return c.Value
	}
	if IsValid(defaultID) {
		return defaultID
	}
	return "futaba"
}

// Sanitize returns id if valid, else fallback.
func Sanitize(id, fallback string) string {
	if IsValid(id) {
		return id
	}
	if IsValid(fallback) {
		return fallback
	}
	return "futaba"
}

// SetCookie writes theme preference (call after Sanitize).
func SetCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// NameFromQuery parses ?theme= or ?name= for set-theme handler.
func NameFromQuery(q string) string {
	return strings.TrimSpace(q)
}
