package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Division struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (s *server) DivisionList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		divisions := []Division{}
		nameQuery := ""
		nameWhere := ""

		q, ok := r.URL.Query()["name"]
		if ok && q[0] != "" {
			nameQuery = q[0]
			nameWhere = "where name like ?"
		}

		query := fmt.Sprintf("select * from divisions %s order by name", nameWhere)
		err := s.DB.Select(&divisions, query, fmt.Sprintf("%%%s%%", nameQuery))
		checkErr(err, "division list")

		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(divisions)
	})
}
