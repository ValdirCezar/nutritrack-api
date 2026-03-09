package service

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// Constantes de segurança
const (
	bcryptCost          = 12              // Custo do bcrypt — equilíbrio entre segurança e performance
	jwtExpiration       = 24 * time.Hour  // Tempo de expiração do token JWT
	minPasswordLen      = 8              // Tamanho mínimo da senha
	codeExpiry          = 10 * time.Minute // Tempo de expiração do código de verificação
	codeCooldown        = 60 * time.Second // Cooldown mínimo entre envios de código
)

// Regex para validação de e-mail (RFC 5322 simplificado)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// AuthService contém a lógica de negócio de autenticação e autorização
type AuthService struct {
	userRepo     *repository.UserRepository
	emailService *EmailService
	jwtSecret    string
}

// NewAuthService cria uma nova instância do serviço de autenticação
func NewAuthService(repo *repository.UserRepository, emailService *EmailService, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:     repo,
		emailService: emailService,
		jwtSecret:    jwtSecret,
	}
}

// Register realiza o cadastro de um novo usuário no sistema.
// O usuário fica como não verificado até confirmar o código enviado por e-mail.
func (s *AuthService) Register(req model.RegisterRequest) error {
	// Valida formato do e-mail
	if !emailRegex.MatchString(req.Email) {
		return errors.New("formato de e-mail inválido")
	}

	// Valida tamanho mínimo da senha
	if len(req.Password) < minPasswordLen {
		return errors.New("a senha deve ter no mínimo 8 caracteres")
	}

	// Verifica se o e-mail já está em uso por um usuário verificado
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if existingUser != nil && existingUser.Verified {
		return errors.New("e-mail já cadastrado")
	}

	// Se existe usuário não verificado, remove para permitir novo cadastro
	if existingUser != nil && !existingUser.Verified {
		_ = s.userRepo.DeleteUserByEmail(req.Email)
	}

	// Gera hash da senha com bcrypt (custo 12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return errors.New("erro ao processar a senha")
	}

	// Cria o objeto do usuário (não verificado)
	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Verified:     false,
	}

	// Persiste no banco de dados
	if err := s.userRepo.CreateUser(user); err != nil {
		if err.Error() == "e-mail já cadastrado" {
			return err
		}
		return errors.New("erro ao criar o usuário")
	}

	// Gera código e salva no banco
	code, err := GenerateCode()
	if err != nil {
		return errors.New("erro ao gerar código de verificação")
	}

	vc := &model.VerificationCode{
		Email:     req.Email,
		Code:      code,
		Type:      "register",
		ExpiresAt: time.Now().Add(codeExpiry),
		Used:      false,
	}

	if err := s.userRepo.SaveVerificationCode(vc); err != nil {
		return errors.New("erro ao salvar código de verificação")
	}

	// Envia e-mail de forma assíncrona para não bloquear o request
	go func() {
		if err := s.emailService.SendVerificationCode(req.Email, code); err != nil {
			log.Printf("Erro ao enviar código de verificação para %s: %v", req.Email, err)
		}
	}()

	return nil
}

// VerifyEmail verifica o código enviado por e-mail e ativa a conta do usuário.
// Retorna token JWT para login automático após verificação.
func (s *AuthService) VerifyEmail(req model.VerifyEmailRequest) (*model.AuthResponse, error) {
	// Busca código válido
	vc, err := s.userRepo.FindValidVerificationCode(req.Email, req.Code, "register")
	if err != nil {
		return nil, errors.New("erro interno do servidor")
	}
	if vc == nil {
		return nil, errors.New("código inválido ou expirado")
	}

	// Marca o código como usado
	_ = s.userRepo.MarkVerificationCodeUsed(vc.ID)

	// Marca o usuário como verificado
	if err := s.userRepo.MarkUserVerified(req.Email); err != nil {
		return nil, errors.New("erro ao verificar o usuário")
	}

	// Busca o usuário para gerar o token
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil {
		return nil, errors.New("erro interno do servidor")
	}

	// Gera token JWT para login automático
	token, err := s.generateJWT(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, errors.New("erro ao gerar token de autenticação")
	}

	return &model.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

// ResendCode reenvia o código de verificação respeitando o cooldown
func (s *AuthService) ResendCode(req model.ResendCodeRequest) error {
	// Valida tipo
	if req.Type != "register" && req.Type != "reset" {
		return errors.New("tipo de código inválido")
	}

	// Valida e-mail
	if !emailRegex.MatchString(req.Email) {
		return errors.New("formato de e-mail inválido")
	}

	// Verifica cooldown — impede spam de códigos
	lastCode, err := s.userRepo.GetLastVerificationCode(req.Email, req.Type)
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if lastCode != nil {
		elapsed := time.Since(lastCode.CreatedAt)
		if elapsed < codeCooldown {
			remaining := int(codeCooldown.Seconds() - elapsed.Seconds())
			return errors.New("aguarde " + formatSeconds(remaining) + " para solicitar um novo código")
		}
	}

	// Para "register", verifica se o usuário existe e está não verificado
	if req.Type == "register" {
		user, err := s.userRepo.FindByEmail(req.Email)
		if err != nil {
			return errors.New("erro interno do servidor")
		}
		if user == nil {
			return errors.New("e-mail não encontrado")
		}
		if user.Verified {
			return errors.New("e-mail já verificado")
		}
	}

	// Para "reset", verifica se o usuário existe e está verificado
	if req.Type == "reset" {
		user, err := s.userRepo.FindByEmail(req.Email)
		if err != nil {
			// Não revelar se o e-mail existe (para reset, retorna sucesso genérico)
			return nil
		}
		if user == nil || !user.Verified {
			// Não revelar se o e-mail existe
			return nil
		}
	}

	// Envia o código
	return s.sendVerificationCode(req.Email, req.Type)
}

// Login autentica o usuário com e-mail e senha, retornando um token JWT.
func (s *AuthService) Login(req model.LoginRequest) (*model.AuthResponse, error) {
	// Busca o usuário pelo e-mail
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("erro interno do servidor")
	}

	// Se o usuário não existir, ainda assim executa o bcrypt para evitar timing attack.
	if user == nil {
		_ = bcrypt.CompareHashAndPassword(
			[]byte("$2a$12$K4YZBRRCvdpSxu3G/MYOOeIQOMiSOvbNqOmFKVhEo2SsepO7sA6hS"),
			[]byte(req.Password),
		)
		return nil, errors.New("e-mail ou senha inválidos")
	}

	// Verifica se o usuário está verificado
	if !user.Verified {
		return nil, errors.New("e-mail não verificado. Verifique sua caixa de entrada.")
	}

	// Compara a senha fornecida com o hash armazenado
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("e-mail ou senha inválidos")
	}

	// Gera token JWT com expiração de 24 horas
	token, err := s.generateJWT(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, errors.New("erro ao gerar token de autenticação")
	}

	return &model.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

// ForgotPassword envia um código de verificação para recuperação de senha.
// IMPORTANTE: Nunca revela se o e-mail existe ou não (prevenção de enumeração).
func (s *AuthService) ForgotPassword(req model.ForgotPasswordRequest) error {
	// Valida formato do e-mail
	if !emailRegex.MatchString(req.Email) {
		return nil // Retorna sem erro para não revelar informações
	}

	// Busca o usuário pelo e-mail
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil || !user.Verified {
		// Retorna sem erro para não revelar existência do e-mail
		return nil
	}

	// Verifica cooldown
	lastCode, err := s.userRepo.GetLastVerificationCode(req.Email, "reset")
	if err == nil && lastCode != nil {
		elapsed := time.Since(lastCode.CreatedAt)
		if elapsed < codeCooldown {
			// Silenciosamente não envia para não revelar informações
			return nil
		}
	}

	// Envia código de recuperação
	if err := s.sendVerificationCode(req.Email, "reset"); err != nil {
		log.Printf("Erro ao enviar código de reset para %s: %v", req.Email, err)
		// Não retorna erro para não revelar informações
	}

	return nil
}

// ResetPassword redefine a senha usando um código de verificação.
func (s *AuthService) ResetPassword(req model.ResetPasswordRequest) error {
	// Valida campos
	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		return errors.New("e-mail, código e nova senha são obrigatórios")
	}

	// Valida tamanho mínimo da nova senha
	if len(req.NewPassword) < minPasswordLen {
		return errors.New("a nova senha deve ter no mínimo 8 caracteres")
	}

	// Busca código de verificação válido
	vc, err := s.userRepo.FindValidVerificationCode(req.Email, req.Code, "reset")
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if vc == nil {
		return errors.New("código inválido ou expirado")
	}

	// Busca o usuário
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil {
		return errors.New("erro interno do servidor")
	}

	// Gera hash da nova senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		return errors.New("erro ao processar a nova senha")
	}

	// Atualiza a senha do usuário
	if err := s.userRepo.UpdatePassword(user.ID, string(hashedPassword)); err != nil {
		return errors.New("erro ao atualizar a senha")
	}

	// Marca o código como usado
	_ = s.userRepo.MarkVerificationCodeUsed(vc.ID)

	return nil
}

// ChangePassword altera a senha do usuário autenticado.
func (s *AuthService) ChangePassword(userID primitive.ObjectID, req model.ChangePasswordRequest) error {
	// Valida tamanho mínimo da nova senha
	if len(req.NewPassword) < minPasswordLen {
		return errors.New("a nova senha deve ter no mínimo 8 caracteres")
	}

	// Busca o usuário no banco de dados
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if user == nil {
		return errors.New("usuário não encontrado")
	}

	// Verifica se a senha atual está correta
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errors.New("senha atual incorreta")
	}

	// Gera hash da nova senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		return errors.New("erro ao processar a nova senha")
	}

	// Atualiza a senha no banco de dados
	if err := s.userRepo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return errors.New("erro ao atualizar a senha")
	}

	return nil
}

// ChangeEmail altera o e-mail do usuário autenticado.
func (s *AuthService) ChangeEmail(userID primitive.ObjectID, req model.ChangeEmailRequest) error {
	// Valida formato do novo e-mail
	if !emailRegex.MatchString(req.NewEmail) {
		return errors.New("formato de e-mail inválido")
	}

	// Busca o usuário no banco de dados
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if user == nil {
		return errors.New("usuário não encontrado")
	}

	// Verifica se a senha está correta (confirmação de identidade)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return errors.New("senha incorreta")
	}

	// Verifica se o novo e-mail já está em uso
	exists, err := s.userRepo.EmailExists(req.NewEmail)
	if err != nil {
		return errors.New("erro interno do servidor")
	}
	if exists {
		return errors.New("e-mail já está em uso")
	}

	// Atualiza o e-mail no banco de dados
	if err := s.userRepo.UpdateEmail(userID, req.NewEmail); err != nil {
		if err.Error() == "e-mail já está em uso" {
			return err
		}
		return errors.New("erro ao atualizar o e-mail")
	}

	return nil
}

// GetUserByID busca um usuário pelo ID. Utilizado pelo handler de perfil.
func (s *AuthService) GetUserByID(userID primitive.ObjectID) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

// sendVerificationCode gera e envia um código de verificação por e-mail.
// O envio do e-mail é feito de forma assíncrona para não bloquear o request.
func (s *AuthService) sendVerificationCode(email, codeType string) error {
	// Gera código de 6 dígitos
	code, err := GenerateCode()
	if err != nil {
		return errors.New("erro ao gerar código de verificação")
	}

	// Salva o código no banco de dados
	vc := &model.VerificationCode{
		Email:     email,
		Code:      code,
		Type:      codeType,
		ExpiresAt: time.Now().Add(codeExpiry),
		Used:      false,
	}

	if err := s.userRepo.SaveVerificationCode(vc); err != nil {
		return errors.New("erro ao salvar código de verificação")
	}

	// Envia o código por e-mail de forma assíncrona
	go func() {
		var sendErr error
		if codeType == "register" {
			sendErr = s.emailService.SendVerificationCode(email, code)
		} else {
			sendErr = s.emailService.SendPasswordResetCode(email, code)
		}
		if sendErr != nil {
			log.Printf("Erro ao enviar e-mail para %s (tipo: %s): %v", email, codeType, sendErr)
		}
	}()

	return nil
}

// generateJWT gera um token JWT assinado com as claims do usuário.
func (s *AuthService) generateJWT(userID, email string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(jwtExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// formatSeconds formata segundos para exibição amigável
func formatSeconds(seconds int) string {
	if seconds <= 1 {
		return "1 segundo"
	}
	if seconds < 60 {
		return fmt.Sprintf("%d segundos", seconds)
	}
	return "1 minuto"
}
