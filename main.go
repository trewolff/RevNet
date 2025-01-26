package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	db        *sqlx.DB
	upgrader  = websocket.Upgrader{}
	jwtSecret = []byte("your_secret_key") // Change this to a secure key
)

type User struct {
	ID       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"-"`
}

type Message struct {
	User    string `json:"user"`
	Content string `json:"content"`
}

// Initialize the database connection
func initDB() {
	var err error
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)
	db, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalln(err)
	}
}

// JWT Middleware
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error reading JSON:", err)
			break
		}
		log.Printf("Received message: %+v\n", msg)
		// Echo the message back
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Println("Error writing JSON:", err)
			break
		}
	}
}

// RESTful API to get users
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	var users []User
	err := db.Select(&users, "SELECT id, username FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(users)
}

// Main function
func main() {
	fmt.Println("Starting app")
	initDB()
	defer db.Close()
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/api/users", jwtMiddleware(http.HandlerFunc(getUsersHandler)))

	fmt.Println("Starting server")
	// Serve static files (HTML, CSS)
	http.Handle("/", http.FileServer(http.Dir("./static"))) // Place your HTML and CSS in the static directory

	fmt.Println("Server started at :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
