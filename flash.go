package main

import (
	"net/http"
)

// ResponseWriter is needed to set a session cookie
func (s *server) SetMessage(w http.ResponseWriter, req *http.Request, msg string) error {
	ss, ok := s.getSession(req)
	if ok {
		ss.Meta["message"] = msg
	} else {
		return s.Mgr.Init(
			w,
			req,
			"0",
			func(m map[string]string) { m["message"] = msg })
	}
	return nil
}

func (s *server) GetMessage(req *http.Request) (msg string) {
	ss, ok := s.getSession(req)
	if ok {
		msg = ss.Meta["message"]
		// delete after use
		ss.Meta["message"] = ""
	}
	return
}
