package handler

import (
	"encoding/json"
	"net/http"
)

// respondWithError envia uma resposta de erro em formato JSON padronizado.
// Utilizado pelos handlers de perfil, refeições e dashboard.
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// respondWithJSON envia uma resposta de sucesso em formato JSON.
// Encapsula o payload em {"data": ...} para manter contrato consistente com o frontend.
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": payload,
	})
}
