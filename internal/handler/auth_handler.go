package handler

import (
	"encoding/json"
	"net/http"

	"github.com/valdircezar/nutritrack-api/internal/middleware"
	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/service"
)

// AuthHandler gerencia as rotas HTTP de autenticação e gerenciamento de conta
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler cria uma nova instância do handler de autenticação
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register trata requisições POST /api/auth/register
// Cadastra um novo usuário (não verificado) e envia código de verificação por e-mail.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	// Valida campos obrigatórios
	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail e senha são obrigatórios")
		return
	}

	// Delega para o serviço de autenticação
	err := h.authService.Register(req)
	if err != nil {
		statusCode := http.StatusBadRequest
		switch err.Error() {
		case "erro interno do servidor", "erro ao criar o usuário", "erro ao processar a senha", "erro ao enviar código de verificação":
			statusCode = http.StatusInternalServerError
		case "e-mail já cadastrado":
			statusCode = http.StatusConflict
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusCreated, "código de verificação enviado para o e-mail", nil)
}

// VerifyEmail trata requisições POST /api/auth/verify-email
// Verifica o código de e-mail e ativa a conta, retornando token JWT.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	// Valida campos obrigatórios
	if req.Email == "" || req.Code == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail e código são obrigatórios")
		return
	}

	// Delega para o serviço de autenticação
	response, err := h.authService.VerifyEmail(req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "erro interno do servidor" || err.Error() == "erro ao verificar o usuário" {
			statusCode = http.StatusInternalServerError
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "e-mail verificado com sucesso", response)
}

// ResendCode trata requisições POST /api/auth/resend-code
// Reenvia o código de verificação respeitando o cooldown de 60 segundos.
func (h *AuthHandler) ResendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.ResendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	// Valida campos obrigatórios
	if req.Email == "" || req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail e tipo são obrigatórios")
		return
	}

	// Delega para o serviço
	err := h.authService.ResendCode(req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "erro interno do servidor" {
			statusCode = http.StatusInternalServerError
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "código reenviado com sucesso", nil)
}

// Login trata requisições POST /api/auth/login
// Autentica o usuário e retorna token JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	// Valida campos obrigatórios
	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail e senha são obrigatórios")
		return
	}

	// Delega para o serviço de autenticação
	response, err := h.authService.Login(req)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if err.Error() == "erro interno do servidor" || err.Error() == "erro ao gerar token de autenticação" {
			statusCode = http.StatusInternalServerError
		}
		if err.Error() == "e-mail não verificado. Verifique sua caixa de entrada." {
			statusCode = http.StatusForbidden
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "login realizado com sucesso", response)
}

// ChangePassword trata requisições PUT /api/auth/password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "usuário não autenticado")
		return
	}

	var req model.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	if req.OldPassword == "" || req.NewPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "senha atual e nova senha são obrigatórias")
		return
	}

	if err := h.authService.ChangePassword(userID, req); err != nil {
		statusCode := http.StatusBadRequest
		switch err.Error() {
		case "senha atual incorreta":
			statusCode = http.StatusUnauthorized
		case "usuário não encontrado":
			statusCode = http.StatusNotFound
		case "erro interno do servidor", "erro ao processar a nova senha", "erro ao atualizar a senha":
			statusCode = http.StatusInternalServerError
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "senha alterada com sucesso", nil)
}

// ForgotPassword trata requisições POST /api/auth/forgot-password
// Envia um código de recuperação. Nunca revela se o e-mail existe.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	if req.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail é obrigatório")
		return
	}

	// Delega para o serviço — retorna mensagem genérica independente do resultado
	_ = h.authService.ForgotPassword(req)

	writeJSONSuccess(w, http.StatusOK, "se o e-mail estiver cadastrado, você receberá um código de recuperação", nil)
}

// ResetPassword trata requisições POST /api/auth/reset-password
// Redefine a senha usando o código de verificação enviado por e-mail.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	var req model.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "e-mail, código e nova senha são obrigatórios")
		return
	}

	if err := h.authService.ResetPassword(req); err != nil {
		statusCode := http.StatusBadRequest
		switch err.Error() {
		case "erro interno do servidor", "erro ao processar a nova senha", "erro ao atualizar a senha":
			statusCode = http.StatusInternalServerError
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "senha redefinida com sucesso", nil)
}

// ChangeEmail trata requisições PUT /api/auth/email
func (h *AuthHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "usuário não autenticado")
		return
	}

	var req model.ChangeEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	defer r.Body.Close()

	if req.NewEmail == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "novo e-mail e senha são obrigatórios")
		return
	}

	if err := h.authService.ChangeEmail(userID, req); err != nil {
		statusCode := http.StatusBadRequest
		switch err.Error() {
		case "senha incorreta":
			statusCode = http.StatusUnauthorized
		case "usuário não encontrado":
			statusCode = http.StatusNotFound
		case "e-mail já está em uso":
			statusCode = http.StatusConflict
		case "erro interno do servidor", "erro ao atualizar o e-mail":
			statusCode = http.StatusInternalServerError
		}
		writeJSONError(w, statusCode, err.Error())
		return
	}

	writeJSONSuccess(w, http.StatusOK, "e-mail alterado com sucesso", nil)
}

// GetProfile trata requisições GET /api/auth/me
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}

	userID, err := middleware.GetUserID(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "usuário não autenticado")
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "erro ao buscar perfil do usuário")
		return
	}
	if user == nil {
		writeJSONError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}

	writeJSONSuccess(w, http.StatusOK, "perfil do usuário", user)
}

// writeJSONError envia uma resposta de erro em formato JSON padronizado
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// writeJSONSuccess envia uma resposta de sucesso em formato JSON padronizado
func writeJSONSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"message": message,
	}
	if data != nil {
		response["data"] = data
	}

	json.NewEncoder(w).Encode(response)
}
