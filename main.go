package main

import (
	"database/sql"
	_ "fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Username string
	Password string
	Email    string
	Phone    string
}

var db *sql.DB

func main() {
	// Установка соединения с базой данных SQLite3
	var err error
	db, err = sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создание таблицы пользователей, если она не существует
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
		port = "8080" // Порт по умолчанию, если переменная окружения PORT не установлена
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	log.Printf("Server started on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Проверка авторизации пользователя
	cookie, err := r.Cookie("session_token")
	if err != nil || !isSessionValid(cookie.Value) {
		// Перенаправление на страницу авторизации
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Получение данных пользователя из базы данных
	username := getUsernameFromSession(cookie.Value)
	user := getUser(username)

	// Загрузка шаблона HTML
	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Отображение данных пользователя на странице
	err = tmpl.Execute(w, user)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Получение данных из формы регистрации
		username := r.FormValue("username")
		password := r.FormValue("password")
		email := r.FormValue("email")
		phone := r.FormValue("phone")

		// Проверка, что пользователь с таким именем не существует
		if userExists(username) {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		// Вставка нового пользователя в базу данных
		err := insertUser(username, password, email, phone)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу авторизации
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		// Загрузка шаблона HTML
		tmpl, err := template.ParseFiles("templates/register.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Отображение страницы регистрации
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		/*
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			err = tmpl.Execute(w, nil)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		*/
		// Получение данных из формы авторизации
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Проверка соответствия имени пользователя и пароля в базе данных
		if !authenticateUser(username, password) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Создание новой сессии и установка куки
		sessionToken := createSession(username)
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)

		// Перенаправление на главную страницу
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		// Загрузка шаблона HTML
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Отображение страницы авторизации
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// Проверка, что пользователь с заданным именем существует
func userExists(username string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		log.Println(err)
		return false
	}
	return count > 0
}

// Вставка нового пользователя в базу данных
func insertUser(username, password, email, phone string) error {
	_, err := db.Exec("INSERT INTO users (username, password, email, phone) VALUES (?, ?, ?, ?)",
		username, password, email, phone)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Проверка соответствия имени пользователя и пароля в базе данных
func authenticateUser(username, password string) bool {
	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
	if err != nil {
		log.Println(err)
		return false
	}
	return password == storedPassword
}

// Создание новой сессии для пользователя
func createSession(username string) string {
	// Здесь можно реализовать генерацию случайного токена
	// и сохранение его в базе данных для связки с пользователем
	// В данном примере будет использован простой вариант сессии,
	// где токеном является имя пользователя
	return username
}

// Проверка валидности сессии
func isSessionValid(sessionToken string) bool {
	// Здесь можно реализовать проверку валидности токена
	// и его связку с пользователем в базе данных
	// В данном примере сессия считается валидной, если токен равен имени пользователя
	return sessionToken == getUsernameFromSession(sessionToken)
}

// Получение имени пользователя из сессии
func getUsernameFromSession(sessionToken string) string {
	// Здесь можно реализовать получение имени пользователя
	// по заданному токену из базы данных
	// В данном примере сессионный токен считается именем пользователя
	return sessionToken
}

// Получение данных пользователя из базы данных
func getUser(username string) *User {
	var user User
	err := db.QueryRow("SELECT * FROM users WHERE username = ?", username).Scan(&user.Username, &user.Password, &user.Email, &user.Phone)
	if err != nil {
		log.Println(err)
	}
	return &user
}
