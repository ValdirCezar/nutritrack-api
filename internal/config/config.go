package config

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	MongoURI    string
	MongoDB     string
	JWTSecret   string
	OpenAIKey   string
	OpenAIModel string
	ServerPort  string
	CORSOrigin  string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

// Load carrega as configurações a partir de variáveis de ambiente.
// SEGURANCA: Exige que JWT_SECRET seja definido explicitamente para evitar uso de segredo padrão.
func Load() *Config {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("FATAL: variável de ambiente JWT_SECRET não está definida. Defina um segredo forte para assinar tokens JWT.")
	}
	if len(jwtSecret) < 32 {
		log.Println("AVISO: JWT_SECRET tem menos de 32 caracteres. Considere usar um segredo mais longo para maior segurança.")
	}

	return &Config{
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:     getEnv("MONGO_DATABASE", "nutritrack"),
		JWTSecret:   jwtSecret,
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
		OpenAIModel: getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		ServerPort:  getPort(),
		CORSOrigin:  getEnv("CORS_ORIGIN", "http://localhost:4200"),
		SMTPHost:    getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:    getEnv("SMTP_PORT", "587"),
		SMTPUser:    getEnv("SMTP_USER", ""),
		SMTPPass:    getEnv("SMTP_PASS", ""),
		SMTPFrom:    getEnv("SMTP_FROM", ""),
	}
}

// ConnectMongo estabelece conexão com o MongoDB e retorna o client e o database
func ConnectMongo(cfg *Config) (*mongo.Client, *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("Erro ao conectar no MongoDB: %v", err)
	}

	// Verifica se a conexão está ativa
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Erro ao fazer ping no MongoDB: %v", err)
	}

	log.Println("Conectado ao MongoDB com sucesso")
	return client, client.Database(cfg.MongoDB)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// getPort retorna a porta do servidor.
// Railway injeta PORT automaticamente; SERVER_PORT serve como fallback para desenvolvimento local.
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		return port
	}
	return "8080"
}
