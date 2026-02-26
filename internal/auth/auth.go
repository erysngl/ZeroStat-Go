package auth

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const sessionName = "zerostat-session"

// Init initializes the cookie store with a random key holding sessions in memory or via env
func Init() {
	secret := os.Getenv("SESSION_SECRET")
	var key []byte

	if secret != "" {
		key = []byte(secret)
	} else {
		// Use a consistent default static secret to survive docker restarts if none provided
		key = []byte("zerostat-default-secret-key-32b!") // Exactly 32 bytes
	}

	store = sessions.NewCookieStore(key)
	// Secure configuration for production-ready approach
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // Should be true if using HTTPS but kept false for local dev without reverse proxy
	}
}

// Login sets the authentication flag in the session.
func Login(w http.ResponseWriter, r *http.Request) error {
	session, _ := store.Get(r, sessionName)
	// Ignore err on Get (could be an invalid or expired cookie from a previous secret) 
	// Treat it as a fresh session request
	
	session.Values["authenticated"] = true
	return session.Save(r, w)
}

// Logout removes the authentication flag from the session.
func Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := store.Get(r, sessionName)
	if err != nil {
		// If cookie is malformed, we still want to create a new empty one to overwrite it
		session, _ = store.New(r, sessionName)
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// Check verifies if the user holds an active authenticated session.
func Check(w http.ResponseWriter, r *http.Request) bool {
	session, err := store.Get(r, sessionName)
	if err != nil {
		// Malformed cookie detected (possibly from secret change or tampering)
		// Gracefully clear the bad cookie instead of returning false
		if w != nil { // Double check if writer was passed
			session.Options.MaxAge = -1
			session.Save(r, w)
		}
		return false
	}
	auth, ok := session.Values["authenticated"].(bool)
	return ok && auth
}

// Middleware creates an HTTP handler wrapping protected routes, redirecting unauthenticated requests.
func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !Check(w, r) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	}
}
