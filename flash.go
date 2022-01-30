package main

import (
	"sync"
)

// TODO: fix race condition where you have multiple non-logged-in users setting
// messages.

type MessageStore struct {
	Mut      sync.Mutex
	Messages map[int]string
}

func NewMessageStore() MessageStore {
	return MessageStore{
		Messages: make(map[int]string),
	}
}

// ResponseWriter is needed to set a session cookie
func (s *server) SetMessage(u player, msg string) {
	s.MsgStore.Mut.Lock()
	defer s.MsgStore.Mut.Unlock()

	s.MsgStore.Messages[u.Id] = msg
}

func (s *server) GetMessage(u player) (msg string) {
	s.MsgStore.Mut.Lock()
	defer s.MsgStore.Mut.Unlock()

	msg = s.MsgStore.Messages[u.Id]
	// delete after use
	s.MsgStore.Messages[u.Id] = ""
	return
}
