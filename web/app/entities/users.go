package entities

import (
	"net/http"
	"syslog-web/data/SQL"
)

func SaveUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Nome de utilizador e palavra-passe são obrigatórios", http.StatusBadRequest)
		return
	}

	// Inserir o novo utilizador na base de dados
	_, err := SQL.DB.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, password)
	if err != nil {
		http.Error(w, "Erro ao criar utilizador: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Utilizador criado com sucesso"))
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")

	if username == "" {
		http.Error(w, "Nome de utilizador é obrigatório", http.StatusBadRequest)
		return
	}

	// Remover o utilizador da base de dados
	_, err := SQL.DB.Exec("DELETE FROM users WHERE username = $1", username)
	if err != nil {
		http.Error(w, "Erro ao remover utilizador: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Utilizador removido com sucesso"))
}