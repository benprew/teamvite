package http

import (
	"net/http"
)

// TODO: fix race condition where you have multiple non-logged-in users setting
// messages.
// Store flash messages in session instead?

// type MessageStore struct {
// 	Mut      sync.Mutex
// 	Messages map[int]string
// }

// func NewMessageStore() MessageStore {
// 	return MessageStore{
// 		Messages: make(map[int]string),
// 	}
// }

// SetFlash sets the flash cookie for the next request to read.
func SetFlash(w http.ResponseWriter, s string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    s,
		Path:     "/",
		HttpOnly: true,
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

// func (s *Server) SetFlashMessage(ctx context.Context, msg string) {
// 	s.MsgStore.Mut.Lock()
// 	defer s.MsgStore.Mut.Unlock()

// 	uid := teamvite.PlayerIDFromContext(ctx)
// 	s.MsgStore.Messages[uid] = msg
// }

// func (s *Server) GetFlashMessage(ctx context.Context) (msg string) {
// 	s.MsgStore.Mut.Lock()
// 	defer s.MsgStore.Mut.Unlock()

// 	uid := teamvite.PlayerIDFromContext(ctx)
// 	msg = s.MsgStore.Messages[uid]
// 	// delete after use
// 	s.MsgStore.Messages[uid] = ""
// 	return
// }
