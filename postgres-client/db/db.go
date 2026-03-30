package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type Db struct {
	pDb *pgxpool.Pool
}

type User struct {
	UserID        string `json:"user_id" db:"userid"`
	Username      string `json:"username" db:"username"`
	Phone         string `json:"phone,omitempty" db:"phone"`
	LastSeen      string `json:"last_seen,omitempty" db:"last_seen"`
	StatusMessage string `json:"status_message,omitempty" db:"status_message"`
	CreatedAt     string `json:"created_at" db:"created_at"`
}

func NewDb() (*Db, error) {
	dbConnString := os.Getenv("POSTGRES_CONN_STRING")
	if dbConnString == "" {
		panic(dbConnString)
	}
	db, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		fmt.Println("failed to open the db connection", err.Error())
		return nil, fmt.Errorf("failed to connect database %s", err)
	}
	err = db.Ping(context.Background())
	if err != nil {
		fmt.Println("failed to ping the db connection")
		return nil, fmt.Errorf("failed to connect to the database %s", err)
	}
	fmt.Println("connected to database successfully!")
	return &Db{
		pDb: db,
	}, nil
}

func (db *Db) Disconnect() {
	if db.pDb != nil {
		fmt.Println("disconnecting from the mongodb")
		db.pDb.Close()
	}
}

func (db *Db) GetUserByName(name string) (User, error) {
	if name == "" {
		return User{}, fmt.Errorf("filter can't be empty")
	}
	var userData User
	rows := db.pDb.QueryRow(context.Background(), "select userid, username from users where username = $1", name)
	err := rows.Scan(&userData.UserID, &userData.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return a clean error or a custom "NotFound" type
			return User{}, fmt.Errorf("user not found: %s", name)
		}
		return User{}, fmt.Errorf("failed to retrieve the contents %s", err.Error())
	}
	return userData, nil
}

func (db *Db) AddUserDetails(user User) bool {
	sqlStatement := `
		INSERT INTO users ( username, phone, last_seen,,created_at,status_message)
		VALUES ($1, $2, $3, $4,$5)` // PostgreSQL specific clause to return the ID

	_, err := db.pDb.Exec(context.Background(), sqlStatement, user.Username, user.Phone, user.LastSeen, time.Now().String(), user.StatusMessage)
	if err != nil {
		fmt.Println("failed to add user to the database", err.Error())
		return false
	}
	return true

}

// create a conversation between the sender and receiver
// a call to this would generate a new conversation id
// which needs to be returned to be used for further processing
// need to support queries where if two user ids are given
// find all the conversations between them
// two of these one to return all the groups they are part of
// one another for returning individual conversations between them

func (db *Db) CreateConversation(isGroup bool, groupid string) (string, error) {
	var conversationId string
	sqlstatement := `INSERT INTO conversations(type) VALUES ($1) RETURNING conversation_id`
	groupType := "direct"

	if isGroup && groupid == "" {
		return conversationId, fmt.Errorf("groupid can't be empty for creating group conversations")
	}
	if isGroup {
		groupType = "group"
		sqlstatement = `INSERT INTO conversations(type, groupid) VALUES ($1, $2) RETURNING conversationid`
	}

	if isGroup {
		err := db.pDb.QueryRow(context.Background(), sqlstatement, groupType, groupid).Scan(&conversationId)
		if err != nil {
			fmt.Println("failed to add conversation to the database", err.Error())
			return conversationId, err
		}

	}
	if !isGroup {
		err := db.pDb.QueryRow(context.Background(), sqlstatement, groupType).Scan(&conversationId)
		if err != nil {
			fmt.Println("failed to add conversation to the database", err.Error())
			return conversationId, err
		}
	}
	return conversationId, nil

}

func (db *Db) CreateConversationMembers(conversationId string, userId string) error {
	sqlStatement := `INSERT INTO conversation_members(conversation_id, userid) VALUES($1, $2)`
	_, err := db.pDb.Exec(context.Background(), sqlStatement, conversationId, userId)
	if err != nil {
		fmt.Println("failed to add conversation to the database", err.Error())
		return fmt.Errorf("failed to create a conversation %s", err.Error())
	}
	return nil
}

func (db *Db) FindConvesationsForUser(userid string) ([]string, error) {
	var conversation_id []string
	if userid == "" {
		fmt.Println("userid required to find the conversations")
		return nil, fmt.Errorf("userid required to find the conversations")
	}
	sqlStatment := `SELECT conversation_id FROM conversation_members WHERE userid = $1`
	rows, err := db.pDb.Query(context.Background(), sqlStatment, userid)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations for the user %s", err.Error())
	}
	for rows.Next() {
		var conversationId string
		err := rows.Scan(&conversationId)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversations for the user %s", err.Error())
		}
		conversation_id = append(conversation_id, conversationId)
	}
	return conversation_id, nil
}

func (db *Db) FindConversationBetweenUsers(userid1 string, userid2 string) (string, error) {
	var conversationId string
	if userid1 == "" || userid2 == "" {
		return conversationId, fmt.Errorf("userid required to find the conversations")
	}
	sqlStatment := `SELECT a.conversation_id
    FROM conversation_members a
    JOIN conversation_members b ON a.conversation_id = b.conversation_id
    WHERE a.userid = $1 AND b.userid = $2`
	err := db.pDb.QueryRow(context.Background(), sqlStatment, userid1, userid2).Scan(&conversationId)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("no direct conversations exists between users")
			return "", nil
		}
		return "", err
	}
	return conversationId, nil
}
