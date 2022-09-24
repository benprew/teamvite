package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type routePath struct {
	Model    string
	Id       uint64
	Method   string
	Template string
}

type RouteContext struct {
	Model    interface{}
	Template string
}

func BuildContext(DB *QueryLogger, r *http.Request) (ctx RouteContext, err error) {
	routeInfo, err := buildRouteInfo(r)
	if err != nil {
		return
	}

	ctx.Template = routeInfo.Template

	var nullM interface{}

	switch routeInfo.Model {
	case "player":
		nullM = player{}
		ctx.Model = loadPlayer(DB, routeInfo.Id)
	case "team":
		nullM = team{}
		ctx.Model = loadTeam(DB, routeInfo.Id)
	case "game":
		nullM = game{}
		ctx.Model = loadGame(DB, routeInfo.Id)
	default:
		err = fmt.Errorf("no model defined for: %s", routeInfo.Model)
		return
	}

	// This handles the case where we should have a model (ie show/edit) but
	// don't because it's invalid and the case where we shouldn't have a model
	// (ie list)
	if routeInfo.Id > 0 && ctx.Model == nullM {
		err = fmt.Errorf("no %s found for id: %d", routeInfo.Model, routeInfo.Id)
		return
	}

	return
}

func buildRouteInfo(req *http.Request) (routePath, error) {
	path := req.URL.EscapedPath()
	pieces := strings.Split(path, "/")
	var rStr routePath
	if len(pieces) == 2 {
		rStr = routePath{pieces[1], 0, "list", ""}
	} else if len(pieces) == 3 {
		rStr = routePath{pieces[1], 0, pieces[2], ""}
	} else if len(pieces) == 4 {
		if id, err := strconv.ParseUint(pieces[2], 10, 32); err != nil {
			return routePath{}, fmt.Errorf("id invalid: %s", pieces[2])
		} else {
			rStr = routePath{pieces[1], id, pieces[3], ""}
		}
	} else {
		return routePath{}, fmt.Errorf("route invalid: %s", path)
	}

	template := fmt.Sprintf("views/%s/%s.tmpl", rStr.Model, rStr.Method)
	log.Printf("template: %s\n", template)
	rStr.Template = template

	return rStr, nil
}

func loadPlayer(DB *QueryLogger, id uint64) (p player) {
	if id < 1 {
		return
	}

	err := DB.Get(&p, "select * from players where id = ?", id)
	checkErr(err, "error loading player: ")
	return
}

func loadTeam(DB *QueryLogger, id uint64) (t team) {
	if id < 1 {
		return
	}
	err := DB.Get(&t, "select * from teams where id = ?", id)
	checkErr(err, "error loading team: ")
	return
}

func loadGame(DB *QueryLogger, id uint64) (g game) {
	if id < 1 {
		return
	}

	err := DB.Get(&g, "select * from games where id = ?", id)
	checkErr(err, "error loading game: ")
	return
}
