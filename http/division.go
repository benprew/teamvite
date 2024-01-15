package http

import (
	"encoding/json"
	"log"
	"net/http"

	teamvite "github.com/benprew/teamvite"
)

type divisionListParams struct {
	Divisions []*teamvite.Division
	Q         string // query
}

func (s *Server) DivisionList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nameQuery := r.URL.Query().Get("name")
		divisions, _, err := s.DivisionService.FindDivisions(r.Context(), teamvite.DivisionFilter{Name: nameQuery})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		switch r.Header.Get("Content-type") {
		case "application/json":
			w.Header().Set("Content-Type", JSON)
			json.NewEncoder(w).Encode(divisions)
		default:
			templateParams := divisionListParams{
				Q:         nameQuery,
				Divisions: divisions,
			}
			err = s.RenderTemplate(w, r, teamvite.TemplateFromContext(r.Context()), templateParams)
			if err != nil {
				log.Println(err)
				s.Error(w, r, err)
				return
			}
		}
	})
}
