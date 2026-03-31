package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arunhiremath92/messenger/mongodb-client/db"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ChatMessage struct {
	ConversationId string `json:"conversation_id" bson:"_id,omitempty"`
	Msg            string `json:"body" bson:"body"`
	SenderId       string `json:"sender_userid" bson:"sender_userid"`
}

type Server struct {
	mogodbConn *db.Db
	server     *http.Server
	mux        *http.ServeMux
}

func ParseJsonBody(r *http.Request) (ChatMessage, error) {
	var msg ChatMessage
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return msg, fmt.Errorf("invalid request, missing message object")
	}
	fmt.Println("msg object received", msg)
	return msg, nil
}

func NewServerInstance() *Server {
	db, err := db.NewDb()
	if err != nil {
		panic(err)
	}
	srv := Server{
		mogodbConn: db,
		mux:        http.NewServeMux(),
	}
	srv.server = &http.Server{Addr: ":6000", Handler: srv.mux}
	return &srv
}

// LoggingMiddleware logs the details of the request before passing it to the next handler.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Logic to execute BEFORE the next handler
		log.Printf("recieved request: %s %s", r.Method, r.URL.Path)
		// Call the next handler in the chain
		next.ServeHTTP(w, r)

	})
}

func (srv *Server) StartHttpServer() error {
	fmt.Println("starting the server instance")
	srv.mux.Handle("/obj/message/create", LoggingMiddleware(http.HandlerFunc(srv.MessageCreateHandler)))
	fmt.Println("registering obj/message/create")
	srv.mux.Handle("/obj/message/delete", LoggingMiddleware(http.HandlerFunc(srv.MessageDelete)))
	fmt.Println("registering obj/message/delete")
	srv.mux.Handle("/obj/message/get", LoggingMiddleware(http.HandlerFunc(srv.MessageList)))
	fmt.Println("registering obj/message/get")
	srv.mux.Handle("/", LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})))
	if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("ListenAndServe error: %v\n", err)
		fmt.Println("closing the http-connection")
		return err
	}
	return nil
}

func (srv *Server) StopServer() {
	srv.mogodbConn.Disconnect()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.server.Shutdown(ctx)
}

func (srv *Server) MessageCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	msg, err := ParseJsonBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	var messageDbObj db.Message
	messageDbObj.ConversationID = msg.ConversationId
	messageDbObj.Body = msg.Msg
	messageDbObj.SenderUserID = msg.SenderId
	messageDbObj.CreatedAt = time.Now().String()
	messageDbObj.Attachments = nil
	messageDbObj.IsDeleted = false
	messageDbObj.TTLExpiresAt = time.Now().AddDate(10, 0, 0).String()

	if (messageDbObj.ConversationID == "") || (messageDbObj.SenderUserID == "") || (messageDbObj.Body == "") {
		fmt.Println("conversationid or senderid or body is empty")
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Println("document that will be inserted is ", messageDbObj)
	isInserted := srv.mogodbConn.InsertMessage(messageDbObj)
	if isInserted {
		fmt.Println("the record was inserted in to collections")
	}
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) MessageDelete(w http.ResponseWriter, r *http.Request) {
	msgBody, err := ParseJsonBody(r)
	if err != nil {
		fmt.Println("failed to parse the json object")
		w.WriteHeader(http.StatusBadRequest)
	}
	filter := bson.D{{Key: "conversation_id", Value: msgBody.ConversationId}}
	isInserted, err := srv.mogodbConn.DeleteMessages(filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	if isInserted {
		fmt.Println("the record was deleted from the collections")
	}
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) MessageList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var response []byte
	query := r.URL.Query()
	// 2. Extract specific fields
	convID := query.Get("conversation_id")
	senderID := query.Get("sender_userid")
	filter := bson.D{}

	// Add conversation_id if it exists
	if convID != "" {
		filter = append(filter, bson.E{Key: "conversation_id", Value: convID})
	}
	// Add sender_userid if it exists
	if senderID != "" {
		filter = append(filter, bson.E{Key: "sender_userid", Value: senderID})
	}
	messages := srv.mogodbConn.FindMessages(filter)
	jsonReponse := map[string][]db.Message{"messages": messages}
	response, err := json.Marshal(&jsonReponse)
	if err != nil {
		fmt.Println("failed to unmarshall the response from the server", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux}

	// Listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		server := NewServerInstance()
		err := server.StartHttpServer()
		if err != nil {
			fmt.Println("failed to start the http server", err)
			stop <- os.Interrupt
		}
	}()

	<-stop // Wait for signal

	fmt.Println("received signal instance from the user; shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("Shutdown error:", err)
	}
}
