package http

import (
	"encoding/json"
	"net/http"

	teamvite "github.com/benprew/teamvite"
)

func (s *Server) DivisionList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nameQuery := r.URL.Query().Get("name")
		divisions, _, err := s.DivisionService.FindDivisions(r.Context(), teamvite.DivisionFilter{Name: nameQuery})
		if err != nil {
			s.Error(w, r, err)
			return
		}
		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(divisions)
	})
}
