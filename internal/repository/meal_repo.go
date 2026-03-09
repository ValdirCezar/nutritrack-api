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

// MealRepository gerencia operações de persistência de refeições no MongoDB
type MealRepository struct {
	collection *mongo.Collection
}

// NewMealRepository cria uma nova instância do repositório de refeições
// e configura os índices necessários para consultas eficientes
func NewMealRepository(db *mongo.Database) *MealRepository {
	repo := &MealRepository{
		collection: db.Collection("meals"),
	}

	// Cria índice composto em user_id e date para consultas por data do usuário
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userDateIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "date", Value: 1},
		},
	}
	_, _ = repo.collection.Indexes().CreateOne(ctx, userDateIndex)

	// Índice de ordenação por data de criação para listagem cronológica
	createdAtIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}
	_, _ = repo.collection.Indexes().CreateOne(ctx, createdAtIndex)

	return repo
}

// Create insere uma nova refeição no banco de dados
func (r *MealRepository) Create(meal *model.Meal) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meal.CreatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, meal)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		meal.ID = oid
	}

	return nil
}

// FindByUserIDAndDate busca todas as refeições de um usuário em uma data específica
func (r *MealRepository) FindByUserIDAndDate(userID primitive.ObjectID, date string) ([]model.Meal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userID,
		"date":    date,
	}

	// Ordena por data de criação (mais antiga primeiro)
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var meals []model.Meal
	if err := cursor.All(ctx, &meals); err != nil {
		return nil, err
	}

	// Retorna slice vazia ao invés de nil para JSON consistente
	if meals == nil {
		meals = []model.Meal{}
	}

	return meals, nil
}

// DeleteByID remove uma refeição pelo ID, garantindo que pertence ao usuário
func (r *MealRepository) DeleteByID(mealID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Filtro garante que o usuário só pode deletar suas próprias refeições
	filter := bson.M{
		"_id":     mealID,
		"user_id": userID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("refeição não encontrada ou não pertence ao usuário")
	}

	return nil
}
