package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Season struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (s *server) SeasonList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seasons := []Season{}
		nameQuery := ""
		nameWhere := ""

		q, ok := r.URL.Query()["name"]
		if ok && q[0] != "" {
			nameQuery = q[0]
			nameWhere = "where name like ?"
		}

		query := fmt.Sprintf("select * from seasons %s order by name", nameWhere)
		err := s.DB.Select(&seasons, query, fmt.Sprintf("%%%s%%", nameQuery))
		checkErr(err, "season list")

		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(seasons)
	})
}
