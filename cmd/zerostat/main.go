package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/erysngl/zerostat/internal/alerting"
	"github.com/erysngl/zerostat/internal/auth"
	"github.com/erysngl/zerostat/internal/config"
	"github.com/erysngl/zerostat/internal/handlers"
	"github.com/erysngl/zerostat/internal/i18n"
)

func main() {
	log.Println("Initializing Config...")
	config.Init()
	
	log.Println("Initializing Auth...")
	auth.Init()

	log.Println("Initializing Templates and i18n...")
	i18n.Init()
	// Move up one directory assuming we might be run from root or cmd
	// In production (Docker), we'll ensure templates are in the working directory
	handlers.InitTemplates()

	log.Println("Starting Alerting Engine...")
	alerting.StartEngine()

	cfg := config.Get()
	port := cfg.GetPort()

	mux := http.NewServeMux()

	// Static Files - Protected by auth or public? Let's make static public to serve Tailwind/CSS for login
	fs := http.FileServer(http.Dir(filepath.Join("static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	mux.HandleFunc("/login", handlers.ServeLogin)
	mux.HandleFunc("/logout", handlers.ServeLogout)

	// Protected Routes wrapped in Middleware
	mux.HandleFunc("/", auth.Middleware(handlers.ServeDashboard))
	mux.HandleFunc("/api/stats", auth.Middleware(handlers.ServeStats))
	mux.HandleFunc("/settings", auth.Middleware(handlers.ServeSettings))
	mux.HandleFunc("/settings/test", auth.Middleware(handlers.TestNotification))
	mux.HandleFunc("/automation", auth.Middleware(handlers.ServeAutomation))
	mux.HandleFunc("/automation/add", auth.Middleware(handlers.AddAutomationRule))
	mux.HandleFunc("/automation/toggle", auth.Middleware(handlers.ToggleAutomationRule))
	mux.HandleFunc("/automation/delete", auth.Middleware(handlers.DeleteAutomationRule))
	
	mux.HandleFunc("/tasks", auth.Middleware(handlers.ServeTasks))
	mux.HandleFunc("/tasks/list", auth.Middleware(handlers.ServeTasksList))
	mux.HandleFunc("/tasks/kill", auth.Middleware(handlers.HandleKillProcess))
	mux.HandleFunc("/tasks/stop_container", auth.Middleware(handlers.HandleStopContainer))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Starting ZeroStat on :%s ...", port)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
