package http

import (
	"net/http"
)

// SetFlash sets the flash cookie for the next request to read.
func SetFlash(w http.ResponseWriter, s string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    s,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func LoadFlash(w http.ResponseWriter, r *http.Request) (s string) {
	// Read & clear flash from cookies.
	if cookie, _ := r.Cookie("flash"); cookie != nil {
		SetFlash(w, "")
		s = cookie.Value
	}
	return
}
