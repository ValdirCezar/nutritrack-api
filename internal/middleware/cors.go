package middleware

import (
	"github.com/rs/cors"
)

// SetupCORS configura o middleware de CORS (Cross-Origin Resource Sharing)
// para permitir requisições do frontend Angular.
func SetupCORS(origin string) *cors.Cors {
	return cors.New(cors.Options{
		// Permite requisições da origem configurada (ex: http://localhost:4200)
		AllowedOrigins: []string{origin},

		// Métodos HTTP permitidos
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

		// Headers permitidos nas requisições
		AllowedHeaders: []string{"Authorization", "Content-Type"},

		// Permite envio de cookies e credenciais
		AllowCredentials: true,
	})
}
