package routes

import (
	"net/http"
	middleware "syslog-web/app/middleware"
	tools "syslog-web/app/tools"
	entities "syslog-web/app/entities"
	logs "syslog-web/app/logs"
	settings "syslog-web/app/settings"
	alerts "syslog-web/app/alerts"
)


func SettingsView() http.HandlerFunc {
	view := settings.LoadSettingsView()
	return view
}


type Route interface {
	Path() string
	Handler() http.Handler
	Middleware() *middleware.AuthMiddleware
}

type Routes struct {
	Path     string
	Handler  http.Handler
	Middleware *middleware.AuthMiddleware
}


func (r *Routes) NewRoute(path string, handler http.Handler, middleware *middleware.AuthMiddleware) *Routes {
	return &Routes{
		Path:       path,
		Handler:    handler,
		Middleware: middleware,
	}
}

func (r *Routes) Route() Routes {
	return *r
}

func LoadRoutes() []Routes {
	var routes []Routes

	routes = append(routes, *(&Routes{}).NewRoute("/", http.HandlerFunc((&middleware.AuthMiddleware{}).ServeDashboard), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/logs", http.HandlerFunc((&middleware.AuthMiddleware{}).logs.ServeLogsPartial), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/stats/view", http.HandlerFunc((&middleware.AuthMiddleware{}).ServeStatsView), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/settings/view", http.HandlerFunc((&middleware.AuthMiddleware{}).settings.ServeSettingsView), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/settings/save", http.HandlerFunc((&middleware.AuthMiddleware{}).settings.SaveSettings), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/tools/view", http.HandlerFunc((&middleware.AuthMiddleware{}).tools.ServeToolsView), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/policies/view", http.HandlerFunc((&middleware.AuthMiddleware{}).settings.ServePoliciesView), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/policies/save", http.HandlerFunc((&middleware.AuthMiddleware{}).settings.SavePolicies), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/alerts/view", http.HandlerFunc((&middleware.AuthMiddleware{}).alerts.ServeAlertsView), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/alerts/save", http.HandlerFunc((&middleware.AuthMiddleware{}).alerts.SaveAlertRule), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/alerts/delete", http.HandlerFunc((&middleware.AuthMiddleware{}).alerts.DeleteAlertRule), (&middleware.AuthMiddleware{}).Middleware()))

	// NOVO: Rotas de Gestão de Utilizadores
	routes = append(routes, *(&Routes{}).NewRoute("/users/save", http.HandlerFunc((&middleware.AuthMiddleware{}).users.SaveUser), (&middleware.AuthMiddleware{}).Middleware()))
	routes = append(routes, *(&Routes{}).NewRoute("/users/delete", http.HandlerFunc((&middleware.AuthMiddleware{}).users.DeleteUser), (&middleware.AuthMiddleware{}).Middleware()))

	return routes
}


