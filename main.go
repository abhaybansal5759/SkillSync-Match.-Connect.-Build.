package main
import(
	"net/http"
	"os"
	"context"
	"fmt"
	"skillsync/internal/db"
    "github.com/joho/godotenv"
	"log"
	"golang.org/x/oauth2"
	"encoding/json"
	"golang.org/x/oauth2/google"

)
var googleOauthConfig *oauth2.Config
	
func main() {
	err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
    db.Init() // Connect to DB

	rows, err := db.Conn.Query(context.Background(),
	`SELECT tablename FROM pg_tables WHERE schemaname='public'`)
if err != nil {
	panic(err)
}
defer rows.Close()

fmt.Println("Tables in database:")
for rows.Next() {
	var tableName string
	if err := rows.Scan(&tableName); err != nil {
		panic(err)
	}
	fmt.Println("-", tableName)
}
http.HandleFunc("/auth/google/login", handleGoogleLogin)
http.HandleFunc("/auth/google/callback", handleGoogleCallback)

fmt.Println("Server started at :8080")
log.Fatal(http.ListenAndServe(":8080", nil))
}	
func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL("randomstate")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}
	log.Println("code ***********", code)

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Token exchange error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch user info
	resp, err := http.Get("https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	type GoogleUser struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Id    string `json:"id"`
	}

	var userInfo GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// // Insert into DB (prevent duplicates)
	// _, err = db.Conn.Exec(context.Background(),
	// 	`INSERT INTO users (name, email) VALUES ($1, $2)
	// 	 ON CONFLICT (email) DO NOTHING`,
	// 	userInfo.Name, userInfo.Email)
	// if err != nil {
	// 	http.Error(w, "DB insert error: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	var existingUser string
err = db.Conn.QueryRow(context.Background(), "SELECT email FROM users WHERE email = $1", userInfo.Email).Scan(&existingUser)

if err != nil {
    if err.Error() == "no rows in result set" {
        // Insert new user
        _, err = db.Conn.Exec(context.Background(),
            "INSERT INTO users (name, email) VALUES ($1, $2)",
            userInfo.Name, userInfo.Email)
        if err != nil {
            http.Error(w, "DB Insert error: "+err.Error(), http.StatusInternalServerError)
            return
        }
        fmt.Fprintln(w, "✅ User saved!")
    } else {
        http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
        return
    }
} else {
    fmt.Fprintln(w, "ℹ️ User already exists.")
	log.Println("************present")
}

	// Show confirmation
	fmt.Fprintf(w, "✅ User saved!\nName: %s\nEmail: %s\nID: %s", userInfo.Name, userInfo.Email, userInfo.Id)
}
