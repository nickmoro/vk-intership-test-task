package repo

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	client     *mongo.Client
	collection *mongo.Collection
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
		client:     client,
		collection: client.Database(dbName).Collection(collectionName),
	}, nil
}

func (r *MongoRepo) Set(userID string, note Note) error {
	filter := bson.M{
		"userID": userID,
	}

	update := bson.M{
		"$push": bson.M{"notes": note},
	}

	_, err := r.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return errors.Wrap(err, "r.collection.UpdateOne")
	}
	return nil
}

func (r *MongoRepo) Get(userID, serviceName string) (Note, error) {
	filter := bson.M{
		"userID":            userID,
		"notes.serviceName": serviceName,
	}

	var user User
	err := r.collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Note{}, ErrNotFound
		}
		return Note{}, errors.Wrap(err, "r.collection.FindOne")
	}
	return user.Notes[0], nil
}

func (r *MongoRepo) Del(userID, serviceName string) error {
	filter := bson.M{
		"userID": userID,
	}

	update := bson.M{
		"$pull": bson.M{
			"notes": bson.M{
				"serviceName": serviceName,
			},
		},
	}

	_, err := r.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return errors.Wrap(err, "r.collection.UpdateOne")
	}
	return nil
}
