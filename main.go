package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
)

var ctx = context.Background()
var client *redis.Client

// User structure to hold username and wins
type User struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
}

func init() {
	// Initialize the Redis client
	client = redis.NewClient(&redis.Options{
		Addr:     "redis-18100.c212.ap-south-1-1.ec2.cloud.redislabs.com:18100",
		Password: "ZGYYfdHk8tb76LZjdhFzmzwRXantEL4O",
		DB:       0,
	})
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v\n", err)
	} else {
		fmt.Printf("Connected to Redis: %s\n", pong)
	}
}

func main() {
	r := mux.NewRouter()

	// Register routes
	r.HandleFunc("/api/register", registerHandler).Methods("POST")
	r.HandleFunc("/api/win", winHandler).Methods("POST")
	r.HandleFunc("/api/leaderboard", leaderboardHandler).Methods("GET")

	// Start HTTP server
	fmt.Println("Server is running on port 10000")
	http.ListenAndServe(":10000", r)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	exists := client.Exists(ctx, user.Username).Val()

	if exists == 0 {
		// User does not exist, initialize wins to 0
		user.Wins = 0

		// Store user in Redis
		if err := client.Set(ctx, user.Username, strconv.Itoa(user.Wins), 0).Err(); err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "User %s registered successfully", user.Username)
	}
}

func winHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch current wins and increment
	Wins, err := client.Get(ctx, user.Username).Result()
	if err == redis.Nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "Error fetching user data", http.StatusInternalServerError)
		return
	}

	currentWins, err := strconv.Atoi(Wins)
	if err != nil {
		http.Error(w, "Error converting wins", http.StatusInternalServerError)
		return
	}

	// Increment and update user wins in Redis
	newWins := currentWins + 1
	//fmt.Printf("New wins count: %d\n", newWins)
	if err := client.Set(ctx, user.Username, strconv.Itoa(newWins), 0).Err(); err != nil {
		http.Error(w, "Failed to update wins", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User %s wins updated successfully", user.Username)
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	var leaderboard []User

	iter := client.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		username := iter.Val()
		wins, err := client.Get(ctx, username).Result()
		if err != nil {
			http.Error(w, "Error fetching leaderboard", http.StatusInternalServerError)
			return
		}

		winsInt, _ := strconv.Atoi(wins)
		user := User{Username: username, Wins: winsInt}
		leaderboard = append(leaderboard, user)
	}

	// Sort the leaderboard based on wins
	sort.Slice(leaderboard, func(i, j int) bool {
		return leaderboard[i].Wins > leaderboard[j].Wins
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}
