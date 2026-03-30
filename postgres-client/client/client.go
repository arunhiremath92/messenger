package main

import (
	"fmt"

	"github.com/arunhiremath92/messenger/postgres-client/db"
)

func main() {
	pDb, err := db.NewDb()
	if err != nil {
		panic(err)
	}
	user1, err := pDb.GetUserByName("alice")
	if err != nil {
		fmt.Println("failed to find the user in the database")

	}
	if user1.Username != "" {
		fmt.Println("username: ", user1.Username, "userid:", user1.UserID)
	}

	user2, err := pDb.GetUserByName("bob")
	if err != nil {
		fmt.Println("failed to find the user in the database")

	}
	if user1.Username != "" {
		fmt.Println("username: ", user2.Username, "userid:", user2.UserID)
	}

	conversation_id_old, err := pDb.FindConversationBetweenUsers(user1.UserID, user2.UserID)
	if err != nil {
		fmt.Println("failed to find conversations between two users", user1.UserID, user2.UserID)
	}
	if conversation_id_old != "" {
		fmt.Println("conversations for the users already exist, no need to create new sets", conversation_id_old)
	}
	if conversation_id_old == "" {
		conversation_id, err := pDb.CreateConversation(false, "")
		if err != nil {
			fmt.Println("failed to create a conversation")
		}
		if conversation_id != "" {
			fmt.Println("conversation id: ", conversation_id)
			err = pDb.CreateConversationMembers(conversation_id, user1.UserID)
			if err != nil {
				fmt.Println("failed to create a conversation entry for the user", user1.UserID)
			}
			err = pDb.CreateConversationMembers(conversation_id, user2.UserID)
			if err != nil {
				fmt.Println("failed to create a conversation entry for the user", user2.UserID)
			}
		}

		conversations, err := pDb.FindConvesationsForUser(user1.UserID)
		if err != nil {
			fmt.Println("failed to retrieive the conversations for user", user2.UserID)
		}
		fmt.Println("conversations user", user1.UserID, " is part of ", conversations)

		conversations_id_2, err := pDb.FindConversationBetweenUsers(user1.UserID, user2.UserID)
		if err != nil {
			fmt.Println("failed to find conversations between two users", user1.UserID, user2.UserID)
		}
		if conversation_id != conversations_id_2 {
			fmt.Println("the conversations created earlier between two users is not right")
		}

	}
	pDb.Disconnect()
}
