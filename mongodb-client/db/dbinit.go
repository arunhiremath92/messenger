package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func SetupDb(mDb *mongo.Client) error {
	database := mDb.Database(DBNAME)
	jsonSchema := bson.M{
		"bsonType": "object",
		"required": []string{"conversation_id", "sender_userid", "body", "created_at"},
		"properties": bson.M{
			"conversation_id": bson.M{"bsonType": "string"},
			"sender_userid":   bson.M{"bsonType": "string"},
			"body":            bson.M{"bsonType": "string"},
			"is_deleted":      bson.M{"bsonType": "bool"},
			"attachments": bson.M{
				"bsonType": "array",
				"items": bson.M{
					"bsonType": "object",
					"required": []string{"url"},
					"properties": bson.M{
						"url":  bson.M{"bsonType": "string"},
						"size": bson.M{"bsonType": "long"},
					},
				},
			},
		},
	}
	opts := options.CreateCollection().SetValidator(bson.M{"$jsonSchema": jsonSchema})
	// CreateCollection will fail if it already exists, so check first or use a 'Try' block
	err := database.CreateCollection(context.TODO(), COLLECTIONS, opts)
	if err != nil {
		// Check if it's a "NamespaceExists" error (code 48)
		if cmdErr, ok := err.(mongo.CommandError); ok && cmdErr.Code == 48 {
			fmt.Println("Collection 'messages' already exists, skipping creation.")
			return nil
		}
		// It's a real error (like a timeout or auth failure)
		return fmt.Errorf("failed to create collection: %w", err)
	}

	fmt.Println("Collection 'messages' created with schema validation.")
	return nil
}
