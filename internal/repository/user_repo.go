package repository

import (
	"context"
	"errors"
	"time"

	"github.com/valdircezar/nutritrack-api/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository gerencia operações de persistência de usuários no MongoDB
type UserRepository struct {
	collection           *mongo.Collection
	tokenCollection      *mongo.Collection
	verifyCodeCollection *mongo.Collection
}

// NewUserRepository cria uma nova instância do repositório e configura os índices necessários
func NewUserRepository(db *mongo.Database) *UserRepository {
	repo := &UserRepository{
		collection:           db.Collection("users"),
		tokenCollection:      db.Collection("password_reset_tokens"),
		verifyCodeCollection: db.Collection("verification_codes"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Índice único no campo email para garantir unicidade
	emailIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, _ = repo.collection.Indexes().CreateOne(ctx, emailIndex)

	// Índice no campo token para buscas rápidas de reset de senha
	tokenIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "token", Value: 1}},
	}
	_, _ = repo.tokenCollection.Indexes().CreateOne(ctx, tokenIndex)

	// Índice TTL para limpeza automática de tokens expirados (expira 24h após criação)
	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(86400),
	}
	_, _ = repo.tokenCollection.Indexes().CreateOne(ctx, ttlIndex)

	// Índice composto para busca de códigos de verificação (email + type)
	verifyIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
			{Key: "type", Value: 1},
		},
	}
	_, _ = repo.verifyCodeCollection.Indexes().CreateOne(ctx, verifyIndex)

	// Índice TTL para limpeza automática de códigos expirados (30 minutos)
	verifyTTL := mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(1800),
	}
	_, _ = repo.verifyCodeCollection.Indexes().CreateOne(ctx, verifyTTL)

	return repo
}

// CreateUser insere um novo usuário no banco de dados.
// Retorna erro se o e-mail já estiver cadastrado (violação de índice único).
func (r *UserRepository) CreateUser(user *model.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		// Verifica se é erro de duplicidade (e-mail já cadastrado)
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("e-mail já cadastrado")
		}
		return err
	}

	// Atribui o ID gerado ao usuário
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	}

	return nil
}

// FindByEmail busca um usuário pelo endereço de e-mail
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// FindByID busca um usuário pelo ID (ObjectID do MongoDB)
func (r *UserRepository) FindByID(id primitive.ObjectID) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// UpdatePassword atualiza o hash da senha de um usuário
func (r *UserRepository) UpdatePassword(userID primitive.ObjectID, newHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"password_hash": newHash,
			"updated_at":    time.Now(),
		},
	}

	result, err := r.collection.UpdateByID(ctx, userID, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("usuário não encontrado")
	}

	return nil
}

// UpdateEmail atualiza o endereço de e-mail de um usuário
func (r *UserRepository) UpdateEmail(userID primitive.ObjectID, newEmail string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"email":      newEmail,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateByID(ctx, userID, update)
	if err != nil {
		// Verifica se é erro de duplicidade (novo e-mail já em uso)
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("e-mail já está em uso")
		}
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("usuário não encontrado")
	}

	return nil
}

// EmailExists verifica se um e-mail já está cadastrado no sistema
func (r *UserRepository) EmailExists(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// MarkUserVerified marca o usuário como verificado (e-mail confirmado)
func (r *UserRepository) MarkUserVerified(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"verified":   true,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"email": email}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("usuário não encontrado")
	}

	return nil
}

// SaveResetToken salva um token de recuperação de senha no banco de dados
func (r *UserRepository) SaveResetToken(token *model.PasswordResetToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token.CreatedAt = time.Now()

	result, err := r.tokenCollection.InsertOne(ctx, token)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		token.ID = oid
	}

	return nil
}

// FindResetToken busca um token de recuperação de senha pelo valor do token
func (r *UserRepository) FindResetToken(token string) (*model.PasswordResetToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resetToken model.PasswordResetToken
	err := r.tokenCollection.FindOne(ctx, bson.M{"token": token}).Decode(&resetToken)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &resetToken, nil
}

// MarkResetTokenUsed marca um token de recuperação como utilizado (uso único)
func (r *UserRepository) MarkResetTokenUsed(tokenID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"used": true,
		},
	}

	_, err := r.tokenCollection.UpdateByID(ctx, tokenID, update)
	return err
}

// --- Métodos para Códigos de Verificação ---

// SaveVerificationCode salva um código de verificação, invalidando códigos anteriores do mesmo tipo/email
func (r *UserRepository) SaveVerificationCode(vc *model.VerificationCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Invalida códigos anteriores do mesmo e-mail e tipo (garante que só o mais recente é válido)
	_, _ = r.verifyCodeCollection.UpdateMany(ctx,
		bson.M{"email": vc.Email, "type": vc.Type, "used": false},
		bson.M{"$set": bson.M{"used": true}},
	)

	vc.CreatedAt = time.Now()

	result, err := r.verifyCodeCollection.InsertOne(ctx, vc)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		vc.ID = oid
	}

	return nil
}

// FindValidVerificationCode busca um código de verificação válido (não usado e não expirado)
func (r *UserRepository) FindValidVerificationCode(email, code, codeType string) (*model.VerificationCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vc model.VerificationCode
	filter := bson.M{
		"email":      email,
		"code":       code,
		"type":       codeType,
		"used":       false,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	err := r.verifyCodeCollection.FindOne(ctx, filter).Decode(&vc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &vc, nil
}

// MarkVerificationCodeUsed marca um código de verificação como usado
func (r *UserRepository) MarkVerificationCodeUsed(codeID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{"used": true},
	}

	_, err := r.verifyCodeCollection.UpdateByID(ctx, codeID, update)
	return err
}

// GetLastVerificationCode busca o último código enviado para um e-mail e tipo (para cooldown)
func (r *UserRepository) GetLastVerificationCode(email, codeType string) (*model.VerificationCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vc model.VerificationCode
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	filter := bson.M{
		"email": email,
		"type":  codeType,
	}

	err := r.verifyCodeCollection.FindOne(ctx, filter, opts).Decode(&vc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &vc, nil
}

// DeleteUserByEmail remove um usuário pelo e-mail (usado para limpar cadastros não verificados)
func (r *UserRepository) DeleteUserByEmail(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"email": email})
	return err
}
