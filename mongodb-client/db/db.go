package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	MONGODB_CONN = "mongodb://localhost:27017/"
	COLLECTIONS  = "messages"
	DBNAME       = "messages"
)

type Db struct {
	mDb *mongo.Client
}

type Attachment struct {
	ID       string `json:"id" bson:"id"`
	FileName string `json:"file_name" bson:"file_name"`
	URL      string `json:"url" bson:"url"`
	Size     int64  `json:"size" bson:"size"` // in bytes
}

// Message represents the core database entity
type Message struct {
	// ID is often a hex string in NoSQL or a primary key in SQL
	ID             string       `json:"_id" bson:"_id,omitempty"`
	ConversationID string       `json:"conversation_id" bson:"conversation_id"`
	SenderUserID   string       `json:"sender_userid" bson:"sender_userid"`
	Body           string       `json:"body" bson:"body"`
	Attachments    []Attachment `json:"attachments" bson:"attachments"`
	IsDeleted      bool         `json:"is_deleted" bson:"is_deleted"`
	CreatedAt      string       `json:"created_at" bson:"created_at"`
	TTLExpiresAt   string       `json:"ttl_expires_at" bson:"ttl_expires_at"`
}

func NewDb() (*Db, error) {

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = MONGODB_CONN
	}
	clientOptions := options.Client().ApplyURI(uri).
		SetBSONOptions(&options.BSONOptions{
			ObjectIDAsHexString: true, // This is the "Magic Switch"
		})
	conn, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db %s", err.Error())
	}

	err = SetupDb(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to setup db %s", err.Error())
	}

	return &Db{
		mDb: conn,
	}, nil
}

func (db *Db) Disconnect() {

	ctx, cancelRtn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelRtn()
	if db.mDb != nil {
		fmt.Println("disconnecting from the mongodb")
		err := db.mDb.Disconnect(ctx)
		if err != nil {
			panic(err)
		}
	}
}

func (db *Db) FindMessages(filter bson.D) []Message {
	var results []Message
	colxns := db.mDb.Database(DBNAME).Collection(COLLECTIONS)
	cursor, err := colxns.Find(context.TODO(), filter)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to query for the messages %s", err.Error()))
		return []Message{}
	}
	err = cursor.All(context.TODO(), &results)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse the messages %s", err.Error()))
	}

	for _, result := range results {
		res, _ := bson.MarshalExtJSON(result, false, false)
		fmt.Println(string(res))
	}
	return results
}

func (db *Db) InsertMessage(msg Message) bool {
	coll := db.mDb.Database(DBNAME).Collection(COLLECTIONS)
	result, err := coll.InsertOne(context.TODO(), msg)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to insert the message %s", err.Error()))
		return false
	}
	return result.Acknowledged
}

func (db *Db) InsertMessages(msg []Message) bool {
	coll := db.mDb.Database(DBNAME).Collection(COLLECTIONS)
	result, err := coll.InsertMany(context.TODO(), msg)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to insert the messages %s", err.Error()))
		return false
	}
	return result.Acknowledged
}

func (db *Db) DeleteMessages(filter bson.D) (bool, error) {
	coll := db.mDb.Database(DBNAME).Collection(COLLECTIONS)
	if filter == nil {
		return false, fmt.Errorf("filters can't be empty for this operation")
	}
	deletedRecord, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		return false, fmt.Errorf("failed to delete records- check filter")
	}

	return deletedRecord.Acknowledged, nil
}

func (db *Db) UpdateMessage(filter bson.D, msg Message) bool {

	coll := db.mDb.Database(DBNAME).Collection(COLLECTIONS)
	update := bson.M{
		"$set": msg,
	}

	result, err := coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		fmt.Printf("failed to update the message: %v\n", err)
		return false
	}
	return result.ModifiedCount > 0
}
