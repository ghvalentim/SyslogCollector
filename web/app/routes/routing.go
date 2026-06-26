package router

import (
	"fmt"
	"log"
	"net/http"
	"syslogs_go/web/app/middleware"
)

// Route define a estrutura de uma rota de forma elegante e limpa
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []middleware.Adapter
}

// Server encapsula o Mux (Router)
type Server struct {
	Router *http.ServeMux
}

// apply encadeia os middlewares no Handler (da última para a primeira camada)
func apply(h http.Handler, middlewares ...middleware.Adapter) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Mount regista as rotas no Router do servidor
func (s *Server) Mount(routes []Route) {
	for _, route := range routes {
		handler := apply(route.Handler, route.Middlewares...)
		
		pattern := route.Path
		if route.Method != "" && route.Method != "ANY" {
			pattern = fmt.Sprintf("%s %s", route.Method, route.Path)
		}
		
		s.Router.Handle(pattern, handler)
		log.Printf("[Router] Rota registada: %s", pattern)
	}
}

// InitRoutes configura as rotas, middlewares e arranca o servidor com Graceful Shutdown
