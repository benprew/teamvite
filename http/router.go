package http

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type routePath struct {
	ModelType string
	ID        uint64
	Method    string
	Template  string
}

type RouteContext struct {
	Model    interface{}
	Template string
}

// func (s *Server) BuildContext(c context.Context, path string) (ctx RouteContext, err error) {
// 	routeInfo, err := buildRouteInfo(path)
// 	if err != nil {
// 		return
// 	}

// 	ctx.Template = routeInfo.Template

// 	var nullM interface{}

// 	switch routeInfo.Model {
// 	case "player":
// 		nullM = player{}
// 		// ctx.Model = s.PlayerService.FindPlayerByID(c, routeInfo.ID)
// 	case "team":
// 		nullM = team{}
// 		ctx.Model, err = s.TeamService.FindTeamByID(c, routeInfo.ID)
// 	case "game":
// 		nullM = teamvite.Game{}
// 		ctx.Model, err = s.GameService.FindGameByID(c, routeInfo.ID)
// 	default:
// 		err = fmt.Errorf("no model defined for: %s", routeInfo.Model)
// 		return
// 	}

// 	// This handles the case where we should have a model (ie show/edit) but
// 	// don't because it's invalid and the case where we shouldn't have a model
// 	// (ie list)
// 	if routeInfo.ID > 0 && ctx.Model == nullM {
// 		err = fmt.Errorf("no %s found for id: %d", routeInfo.Model, routeInfo.ID)
// 		return
// 	}

// 	return
// }

func buildRouteInfo(path string) (routePath, error) {
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

	template := fmt.Sprintf("views/%s/%s.tmpl", rStr.ModelType, rStr.Method)
	log.Printf("template: %s\n", template)
	rStr.Template = template

	return rStr, nil
}
