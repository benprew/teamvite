package http

import (
	"encoding/json"
	"net/http"

	teamvite "github.com/benprew/teamvite"
)

type Season struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *Server) SeasonList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nameQuery := r.URL.Query().Get("name")
		seasons, _, err := s.SeasonService.FindSeasons(r.Context(), teamvite.SeasonFilter{Name: nameQuery})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(seasons)
	})
}
