package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User representa um usuário do sistema
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	Name         string             `bson:"name,omitempty" json:"name,omitempty"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Verified     bool               `bson:"verified" json:"verified"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// RegisterRequest dados para cadastro de usuário
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest dados para login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse resposta de autenticação com token JWT
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ChangePasswordRequest dados para alteração de senha
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ForgotPasswordRequest dados para solicitar recuperação de senha
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest dados para redefinir a senha com código de verificação
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// ChangeEmailRequest dados para alteração de e-mail
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

// PasswordResetToken armazena tokens de recuperação de senha
type PasswordResetToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	Used      bool               `bson:"used" json:"used"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// VerificationCode armazena códigos de verificação por e-mail (registro e reset de senha)
type VerificationCode struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" json:"email"`
	Code      string             `bson:"code" json:"code"`
	Type      string             `bson:"type" json:"type"` // "register" ou "reset"
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	Used      bool               `bson:"used" json:"used"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// VerifyEmailRequest dados para verificar o código de e-mail
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ResendCodeRequest dados para reenviar o código de verificação
type ResendCodeRequest struct {
	Email string `json:"email"`
	Type  string `json:"type"` // "register" ou "reset"
}
