package router

import (
 "net/http"
 "syslog-web/app/middleware"
 "log"
 "os"
 "time"
 "context"
 "os/signal"
 "syscall"
)

func InitRoutes() {
	srv := &Server{
		Router: http.NewServeMux(),
	}

	// 1. Configurar Servidor de Ficheiros Estáticos (Tailwind, JS, Imagens)
	// A pasta "./public" deve existir no root de onde o binário Go é executado no Docker
	fileServer := http.StripPrefix("/public/", http.FileServer(http.Dir("./public")))

	// 2. Declaração Centralizada de Rotas
	appRoutes := []Route{
		{
			// Rota Principal: Serve o ficheiro index.html (O seu painel HTMX)
			Method: "GET",
			Path:   "/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				// Evita que requisições a rotas inexistentes caiam no root "/"
				if r.URL.Path != "/" {
					http.NotFound(w, r)
					return
				}
				http.ServeFile(w, r, "./public/index.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			// Rota para os ficheiros estáticos (Assets)
			Method: "GET",
			Path:   "/public/", 
			Handler: fileServer.ServeHTTP,
			Middlewares: []middleware.Adapter{}, // Omitimos logger aqui para não poluir o terminal com pedidos de CSS
		},
		// ==========================================
		// ROTAS DA APLICAÇÃO (HTMX Views)
		// ==========================================
		{
			Method: "GET",
			Path:   "/logs",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/logs.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/settings",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/settings.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/stats",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/stats.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/tools",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/tools.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/alerts",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/alerts.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/policies",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/policies.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			Method: "GET",
			Path:   "/login",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./app/views/login.html")
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
		{
			// Exemplo de rota de API consumida via HTMX
			Method: "GET",
			Path:   "/api/stats",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<div class="badge badge-success">Serviço de Logs Operacional</div>`))
			},
			Middlewares: []middleware.Adapter{middleware.Recovery, middleware.Logger},
		},
	}

	srv.Mount(appRoutes)

	// 3. Configuração da Porta
	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] == ':' {
		port = port[1:] 
	}

	// 4. Configuração Robusta do Servidor HTTP (Prevenção de ataques e lentidão)
	httpServer := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      srv.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second, // Tempo máximo à espera do Nginx
	}

	// 5. Graceful Shutdown (Regra 10: Nunca quebrar abruptamente)
	// Criamos um canal para escutar sinais de paragem do Docker/OS
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Corremos o servidor numa Goroutine (em background)
	go func() {
		log.Printf("Iniciando Servidor HTTP na porta %s (0.0.0.0)...", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ERRO CRÍTICO: Falha fatal no servidor: %v\n", err)
		}
	}()

	// O código bloqueia aqui até receber um sinal do Docker (ex: docker compose stop)
	<-stopChan
	log.Println("Sinal de encerramento recebido. A desligar o servidor graciosamente...")

	// Damos 5 segundos para as requisições pendentes terminarem
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Erro ao forçar o encerramento do servidor: %v", err)
	}

	log.Println("Servidor Web encerrado com segurança. Adeus!")
}



