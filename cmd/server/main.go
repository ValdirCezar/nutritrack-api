package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/valdircezar/nutritrack-api/internal/config"
	"github.com/valdircezar/nutritrack-api/internal/handler"
	"github.com/valdircezar/nutritrack-api/internal/middleware"
	"github.com/valdircezar/nutritrack-api/internal/repository"
	"github.com/valdircezar/nutritrack-api/internal/service"
)

func main() {
	// Carrega variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// Carrega configurações
	cfg := config.Load()

	// Conecta ao MongoDB
	mongoClient, db := config.ConnectMongo(cfg)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("Erro ao desconectar do MongoDB: %v", err)
		}
	}()

	// --- Inicializa Repositories ---
	userRepo := repository.NewUserRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	mealRepo := repository.NewMealRepository(db)
	foodCacheRepo := repository.NewFoodCacheRepository(db)

	// --- Inicializa Services ---
	emailService := service.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	authService := service.NewAuthService(userRepo, emailService, cfg.JWTSecret)
	profileService := service.NewProfileService(profileRepo)
	openaiService := service.NewOpenAIService(cfg.OpenAIKey, cfg.OpenAIModel)
	foodCacheService := service.NewFoodCacheService(foodCacheRepo)
	mealService := service.NewMealService(mealRepo, openaiService, foodCacheService)

	// --- Inicializa Handlers ---
	authHandler := handler.NewAuthHandler(authService)
	profileHandler := handler.NewProfileHandler(profileService)
	mealHandler := handler.NewMealHandler(mealService)
	dashboardHandler := handler.NewDashboardHandler(mealService, profileService)

	// --- Configura Middleware ---
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)
	corsHandler := middleware.SetupCORS(cfg.CORSOrigin)

	// --- Configura Rotas ---
	mux := http.NewServeMux()

	// Health check (usado pelo Render)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Rotas públicas (sem autenticação)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/verify-email", authHandler.VerifyEmail)
	mux.HandleFunc("POST /api/auth/resend-code", authHandler.ResendCode)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/forgot-password", authHandler.ForgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", authHandler.ResetPassword)

	// Rotas protegidas (requerem JWT)
	mux.Handle("GET /api/auth/me", authMiddleware(http.HandlerFunc(authHandler.GetProfile)))
	mux.Handle("PUT /api/auth/password", authMiddleware(http.HandlerFunc(authHandler.ChangePassword)))
	mux.Handle("PUT /api/auth/email", authMiddleware(http.HandlerFunc(authHandler.ChangeEmail)))

	mux.Handle("POST /api/profile", authMiddleware(http.HandlerFunc(profileHandler.CreateOrUpdate)))
	mux.Handle("GET /api/profile", authMiddleware(http.HandlerFunc(profileHandler.Get)))

	mux.Handle("POST /api/meals", authMiddleware(http.HandlerFunc(mealHandler.Register)))
	mux.Handle("GET /api/meals", authMiddleware(http.HandlerFunc(mealHandler.ListByDate)))
	mux.Handle("DELETE /api/meals/{id}", authMiddleware(http.HandlerFunc(mealHandler.Delete)))

	mux.Handle("GET /api/dashboard", authMiddleware(http.HandlerFunc(dashboardHandler.Get)))

	// Aplica CORS em todas as rotas
	finalHandler := corsHandler.Handler(mux)

	// --- Configura Servidor ---
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Maior para aguardar resposta da OpenAI
		IdleTimeout:  120 * time.Second,
	}

	// --- Inicia Servidor com Graceful Shutdown ---
	go func() {
		log.Printf("NutriTrack API rodando na porta %s", cfg.ServerPort)
		log.Printf("CORS habilitado para: %s", cfg.CORSOrigin)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	// Aguarda sinal de interrupção para shutdown gracioso
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Desligando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Erro no shutdown do servidor: %v", err)
	}
	log.Println("Servidor encerrado com sucesso")
}
