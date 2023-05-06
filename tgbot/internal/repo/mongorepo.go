package repo

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	Client     *mongo.Client
	Collection *mongo.Collection
	mutexMap   sync.Map
}

func NewMongoRepo(connectionString, dbName, collectionName string) (NotesRepo, error) {
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(connectionString),
	)

	if err != nil {
		return nil, err
	}

	return &MongoRepo{
		Client:     client,
		Collection: client.Database(dbName).Collection(collectionName),
		mutexMap:   sync.Map{},
	}, nil
}

func (r *MongoRepo) Set(chatID string, note Note) error {
	filter := bson.M{
		"_id": chatID,
	}

	update := bson.M{
		"$push": bson.M{"notes": note},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.Collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return errors.Wrap(err, "r.Collection.UpdateOne")
	}

	return nil
}

func (r *MongoRepo) Get(chatID, serviceName string) (Note, error) {
	filter := bson.M{
		"_id":               chatID,
		"notes.servicename": serviceName,
	}

	var workspace Workspace
	err := r.Collection.FindOne(context.Background(), filter).Decode(&workspace)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Note{}, ErrNotFound
		}
		return Note{}, errors.Wrap(err, "r.Collection.FindOne")
	}

	return workspace.Notes[0], nil
}

func (r *MongoRepo) Del(chatID, serviceName string) error {
	filter := bson.M{
		"_id": chatID,
	}

	update := bson.M{
		"$pull": bson.M{
			"notes": bson.M{
				"servicename": serviceName,
			},
		},
	}

	_, err := r.Collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return errors.Wrap(err, "r.Collection.UpdateOne")
	}

	return nil
}
