package main

import (
	"database/sql"
	_ "fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type User struct {
	Username string
	Password string
	Email    string
	Phone    string
}

var db *sql.DB

func main() {
	// Establish connection to the PostgreSQL database
	connStr := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Create the users table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT,
		email TEXT,
		phone TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if PORT environment variable is not set
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	log.Printf("Server started on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	cookie, err := r.Cookie("session_token")
	if err != nil || !isSessionValid(cookie.Value) {
		// Redirect to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get the user data from the database
	username := getUsernameFromSession(cookie.Value)
	user := getUser(username)

	// Load the HTML template
	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the user data on the page
	err = tmpl.Execute(w, user)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the registration form data
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")
		phone := r.FormValue("phone")

		// Check if the username already exists
		if userExists(username) {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		// Insert the new user into the database
		err := insertUser(username, password, email, phone)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Redirect to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		// Load the HTML template
		tmpl, err := template.ParseFiles("templates/register.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Render the registration page
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the login form data
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Check if the username and password match in the database
		if !authenticateUser(username, password) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Create a new session and set the cookie
		sessionToken := createSession(username)
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)

		// Redirect to the home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		// Load the HTML template
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Render the login page
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// Check if a user with the given username exists
func userExists(username string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", username).Scan(&count)
	if err != nil {
		log.Println(err)
		return false
	}
	return count > 0
}

// Insert a new user into the database
func insertUser(username, password, email, phone string) error {
	_, err := db.Exec("INSERT INTO users (username, password, email, phone) VALUES ($1, $2, $3, $4)",
		username, password, email, phone)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Check if the username and password match in the database
func authenticateUser(username, password string) bool {
	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&storedPassword)
	if err != nil {
		log.Println(err)
		return false
	}
	return password == storedPassword
}

// Create a new session for the user
func createSession(username string) string {
	// Here, you can implement generating a random token
	// and store it in the database to associate with the user
	// In this example, a simple session is used where the token is the username
	return username
}

// Check if the session is valid
func isSessionValid(sessionToken string) bool {
	// Here, you can implement the validation of the token
	// and its association with the user in the database
	// In this example, the session is considered valid if the token is equal to the username
	return sessionToken == getUsernameFromSession(sessionToken)
}

// Get the username from the session token
func getUsernameFromSession(sessionToken string) string {
	// Here, you can implement retrieving the username
	// based on the given token from the database
	// In this example, the session token is considered as the username
	return sessionToken
}

// Get the user data from the database
func getUser(username string) *User {
	var user User
	err := db.QueryRow("SELECT * FROM users WHERE username = $1", username).Scan(&user.Username, &user.Password, &user.Email, &user.Phone)
	if err != nil {
		log.Println(err)
	}
	return &user
}
