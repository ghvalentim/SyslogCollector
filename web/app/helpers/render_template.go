package helpers

import (
	"html/template"
	"log"
	"net/http"
)

// RenderTemplate é uma função auxiliar que carrega e renderiza um template HTML a partir de um caminho especificado.
// Ela recebe o ResponseWriter, o caminho do template e os dados a serem passados para o template.

func RenderTemplate(w http.ResponseWriter, path string, data interface{}) {
	t, err := template.ParseFiles(path)
	if err != nil {
		log.Printf("[ERRO] Não foi possível carregar o template %s: %v", path, err)
		http.Error(w, "Erro interno de apresentação visual.", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}