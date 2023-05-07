package repo

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/yudai/nmutex"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepo struct {
	collection *mongo.Collection
	mu         *nmutex.NamedMutex
}

func NewMongoRepo(mongoURI, dbName, collectionName string) (NotesRepo, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, errors.Wrap(err, "mongo.Connect")
	}

	return &MongoRepo{
		collection: client.Database(dbName).Collection(collectionName),
		mu:         nmutex.New(),
	}, nil
}

func (r *MongoRepo) Set(chatID string, note Note) error {
	// Firstly delete note with such servicename
	filter := bson.M{
		"_id": chatID,
	}

	pullUpdate := bson.M{
		"$pull": bson.M{
			"notes": bson.M{
				"servicename": note.ServiceName,
			},
		},
	}

	pushUpdate := bson.M{
		"$push": bson.M{
			"notes": note,
		},
	}

	pushOpts := options.Update().SetUpsert(true)

	unlocker := r.mu.Lock(chatID)
	defer unlocker()

	// delete
	_, err := r.collection.UpdateOne(context.Background(), filter, pullUpdate)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return errors.Wrap(err, "r.collection.UpdateOne")
	}

	// insert
	_, err = r.collection.UpdateOne(context.Background(), filter, pushUpdate, pushOpts)
	if err != nil {
		return errors.Wrap(err, "r.collection.UpdateOne")
	}

	fmt.Println("Setted")
	return nil
}

func (r *MongoRepo) Get(chatID, serviceName string) (Note, error) {
	filter := bson.M{
		"_id":               chatID,
		"notes.servicename": serviceName,
	}

	unlocker := r.mu.Lock(chatID)
	defer unlocker()

	var workspace Workspace
	err := r.collection.FindOne(context.Background(), filter).Decode(&workspace)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Note{}, ErrNotFound
		}
		return Note{}, errors.Wrap(err, "r.collection.FindOne")
	}

	log.Println("Found workspace =", workspace)
	for _, note := range workspace.Notes {
		if note.ServiceName == serviceName {
			return note, nil
		}
	}
	return Note{}, ErrNotFound
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

	unlocker := r.mu.Lock(chatID)
	defer unlocker()

	updateResult, err := r.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return errors.Wrap(err, "r.collection.UpdateOne")
	}
	if updateResult.ModifiedCount < 1 {
		return ErrNotFound
	}

	return nil
}
