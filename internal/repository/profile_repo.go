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

// ProfileRepository gerencia operações de persistência de perfis no MongoDB
type ProfileRepository struct {
	collection *mongo.Collection
}

// NewProfileRepository cria uma nova instância do repositório de perfis
// e configura o índice único no campo user_id
func NewProfileRepository(db *mongo.Database) *ProfileRepository {
	repo := &ProfileRepository{
		collection: db.Collection("profiles"),
	}

	// Cria índice único no campo user_id (cada usuário tem apenas um perfil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIDIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, _ = repo.collection.Indexes().CreateOne(ctx, userIDIndex)

	return repo
}

// Create insere um novo perfil no banco de dados
func (r *ProfileRepository) Create(profile *model.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, profile)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		profile.ID = oid
	}

	return nil
}

// FindByUserID busca o perfil de um usuário pelo seu ID
func (r *ProfileRepository) FindByUserID(userID primitive.ObjectID) (*model.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var profile model.Profile
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

// Update atualiza um perfil existente no banco de dados
func (r *ProfileRepository) Update(profile *model.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	profile.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"weight":         profile.Weight,
			"height":         profile.Height,
			"age":            profile.Age,
			"sex":            profile.Sex,
			"activity_level": profile.ActivityLevel,
			"goal":           profile.Goal,
			"tmb":            profile.TMB,
			"tdee":           profile.TDEE,
			"daily_calories": profile.DailyCalories,
			"daily_protein":  profile.DailyProtein,
			"daily_carbs":    profile.DailyCarbs,
			"daily_fat":      profile.DailyFat,
			"updated_at":     profile.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"user_id": profile.UserID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("perfil não encontrado")
	}

	return nil
}

// Upsert cria ou atualiza o perfil de um usuário com base no user_id
func (r *ProfileRepository) Upsert(profile *model.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	profile.UpdatedAt = now

	filter := bson.M{"user_id": profile.UserID}
	update := bson.M{
		"$set": bson.M{
			"weight":         profile.Weight,
			"height":         profile.Height,
			"age":            profile.Age,
			"sex":            profile.Sex,
			"activity_level": profile.ActivityLevel,
			"goal":           profile.Goal,
			"target_weight":  profile.TargetWeight,
			"target_weeks":   profile.TargetWeeks,
			"tmb":            profile.TMB,
			"tdee":           profile.TDEE,
			"daily_calories": profile.DailyCalories,
			"daily_protein":  profile.DailyProtein,
			"daily_carbs":    profile.DailyCarbs,
			"daily_fat":      profile.DailyFat,
			"updated_at":     now,
		},
		"$setOnInsert": bson.M{
			"user_id":    profile.UserID,
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	// Se foi um insert, atribui o ID gerado
	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			profile.ID = oid
		}
		profile.CreatedAt = now
	}

	return nil
}
