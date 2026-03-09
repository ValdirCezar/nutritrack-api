package handler

import (
	"encoding/json"
	"net/http"

	"github.com/valdircezar/nutritrack-api/internal/middleware"
	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/service"
)

// ProfileHandler gerencia as requisições HTTP relacionadas ao perfil do usuário
type ProfileHandler struct {
	profileService *service.ProfileService
}

// NewProfileHandler cria uma nova instância do handler de perfis
func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// CreateOrUpdate processa POST /api/profile — cria ou atualiza o perfil do usuário.
// Requer autenticação JWT. Recebe os dados físicos e retorna o perfil com metas calculadas.
func (h *ProfileHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Decodifica o corpo da requisição
	var req model.ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Dados inválidos no corpo da requisição")
		return
	}

	// Chama o serviço para criar ou atualizar o perfil
	profile, err := h.profileService.CreateOrUpdate(userID, req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, profile)
}

// Get processa GET /api/profile — retorna o perfil do usuário autenticado.
// Retorna 404 se o perfil ainda não foi criado.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Busca o perfil do usuário
	profile, err := h.profileService.GetProfile(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar perfil")
		return
	}

	if profile == nil {
		respondWithError(w, http.StatusNotFound, "Perfil não encontrado. Complete o onboarding primeiro.")
		return
	}

	respondWithJSON(w, http.StatusOK, profile)
}
