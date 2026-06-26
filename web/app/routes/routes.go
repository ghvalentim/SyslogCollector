package routes

import (
 "net/http"
)

func InitRoutes() {

	router := http.NewServeMux()

	routes := LoadRoutes()

	for _, route := range routes {
		router.Handle(route.Path, route.Handler)
	}

	http.Handle("/", router)

	
	
}




