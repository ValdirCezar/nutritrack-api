package service

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"strings"
)

// EmailService gerencia o envio de e-mails via SMTP
type EmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

// NewEmailService cria uma nova instância do serviço de e-mail
func NewEmailService(host, port, user, password, from string) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

// GenerateCode gera um código numérico de 6 dígitos criptograficamente seguro
func GenerateCode() (string, error) {
	// Gera número entre 100000 e 999999
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	code := n.Int64() + 100000
	return fmt.Sprintf("%06d", code), nil
}

// SendVerificationCode envia o código de verificação para o e-mail do usuário
func (s *EmailService) SendVerificationCode(toEmail, code string) error {
	subject := "NutriTrack AI - Código de Verificação"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #F5F7FA; padding: 20px;">
  <div style="max-width: 400px; margin: 0 auto; background: #FFFFFF; border-radius: 16px; padding: 32px 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.08);">
    <h1 style="color: #4CAF50; font-size: 24px; text-align: center; margin: 0 0 8px 0;">NutriTrack AI</h1>
    <p style="color: #757575; font-size: 14px; text-align: center; margin: 0 0 24px 0;">Código de verificação</p>

    <div style="background: #F5F7FA; border-radius: 12px; padding: 24px; text-align: center; margin-bottom: 24px;">
      <span style="font-size: 36px; font-weight: 700; letter-spacing: 8px; color: #212121;">%s</span>
    </div>

    <p style="color: #757575; font-size: 13px; text-align: center; margin: 0;">
      Este código expira em <strong>10 minutos</strong>.<br>
      Se você não solicitou este código, ignore este e-mail.
    </p>
  </div>
</body>
</html>`, code)

	return s.sendHTML(toEmail, subject, body)
}

// SendPasswordResetCode envia o código de recuperação de senha
func (s *EmailService) SendPasswordResetCode(toEmail, code string) error {
	subject := "NutriTrack AI - Recuperação de Senha"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #F5F7FA; padding: 20px;">
  <div style="max-width: 400px; margin: 0 auto; background: #FFFFFF; border-radius: 16px; padding: 32px 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.08);">
    <h1 style="color: #4CAF50; font-size: 24px; text-align: center; margin: 0 0 8px 0;">NutriTrack AI</h1>
    <p style="color: #757575; font-size: 14px; text-align: center; margin: 0 0 24px 0;">Recuperação de senha</p>

    <div style="background: #F5F7FA; border-radius: 12px; padding: 24px; text-align: center; margin-bottom: 24px;">
      <span style="font-size: 36px; font-weight: 700; letter-spacing: 8px; color: #212121;">%s</span>
    </div>

    <p style="color: #757575; font-size: 13px; text-align: center; margin: 0;">
      Use este código para redefinir sua senha.<br>
      Ele expira em <strong>10 minutos</strong>.<br>
      Se você não solicitou a recuperação, ignore este e-mail.
    </p>
  </div>
</body>
</html>`, code)

	return s.sendHTML(toEmail, subject, body)
}

// sendHTML envia um e-mail HTML via SMTP
func (s *EmailService) sendHTML(to, subject, htmlBody string) error {
	// Extrai apenas o e-mail do campo "from" (pode ter formato "Nome <email>")
	fromEmail := s.from
	if idx := strings.Index(s.from, "<"); idx != -1 {
		fromEmail = strings.Trim(s.from[idx:], "<>")
	}

	// Monta o cabeçalho e corpo do e-mail
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		s.from, to, subject)

	msg := []byte(headers + htmlBody)

	// Autenticação SMTP
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	// Envia o e-mail
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	err := smtp.SendMail(addr, auth, fromEmail, []string{to}, msg)
	if err != nil {
		log.Printf("Erro ao enviar e-mail para %s: %v", to, err)
		return err
	}

	log.Printf("E-mail enviado com sucesso para %s", to)
	return nil
}
