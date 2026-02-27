package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"strconv"
	"time"

	"github.com/erysngl/zerostat/internal/alerting"
	"github.com/erysngl/zerostat/internal/auth"
	"github.com/erysngl/zerostat/internal/config"
	"github.com/erysngl/zerostat/internal/i18n"
	"github.com/erysngl/zerostat/internal/metrics"
	"sync"
)

var (
	tmplCache      map[string]*template.Template
	loginMu        sync.Mutex
	failedAttempts int
)

// InitTemplates parses templates per page to avoid block name collisions
func InitTemplates() {
	tmplCache = make(map[string]*template.Template)
	pages := []string{"login.html", "dashboard.html", "settings.html", "stats.html", "automation.html", "tasks.html"}

	base := filepath.Join("templates", "base.html")
	stats := filepath.Join("templates", "stats.html")

	for _, page := range pages {
		var files []string
		if page == "stats.html" {
			// Stats only needs its own file and base if it uses it, but let's just parse it directly
			files = append(files, stats)
			tmplCache[page] = template.Must(template.ParseFiles(files...))
		} else if page == "dashboard.html" {
			// dashboard includes stats inside it
			files = append(files, base, filepath.Join("templates", page), stats)
			tmplCache[page] = template.Must(template.ParseFiles(files...))
		} else {
			files = append(files, base, filepath.Join("templates", page))
			tmplCache[page] = template.Must(template.ParseFiles(files...))
		}
	}
}

type PageData struct {
	Theme  string
	Locale string
	T      func(string) template.HTML
	Error  string
	Info   string
	Data   interface{}
}

func getBaseData() PageData {
	cfg := config.Get()
	return PageData{
		Theme:  cfg.GetTheme(),
		Locale: cfg.GetLocale(),
		T:      i18n.TFunc(cfg.GetLocale()),
	}
}

// ServeLogin renders the login page or processes a login attempt
func ServeLogin(w http.ResponseWriter, r *http.Request) {
	if auth.Check(w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := getBaseData()

	if r.Method == http.MethodPost {
		password := r.FormValue("password")

		loginMu.Lock()
		delay := time.Duration(failedAttempts) * 500 * time.Millisecond
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
		loginMu.Unlock()
		
		if delay > 0 {
			time.Sleep(delay)
		}

		if password == config.Get().GetPassword() {
			loginMu.Lock()
			failedAttempts = 0
			loginMu.Unlock()
			err := auth.Login(w, r)
			if err != nil {
				data.Error = "Internal Server Error: " + err.Error()
				tmplCache["login.html"].ExecuteTemplate(w, "base.html", data)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		
		loginMu.Lock()
		failedAttempts++
		loginMu.Unlock()
		
		data.Error = string(data.T("InvalidCreds"))
	}

	tmplCache["login.html"].ExecuteTemplate(w, "base.html", data)
}

// ServeLogout destroys the session
func ServeLogout(w http.ResponseWriter, r *http.Request) {
	auth.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ServeDashboard renders the main layout
func ServeDashboard(w http.ResponseWriter, r *http.Request) {
	data := getBaseData()
	data.Data = metrics.GetFormatted()
	tmplCache["dashboard.html"].ExecuteTemplate(w, "base.html", data)
}

// ServeStats serves just the stats snippet for HTMX polling
func ServeStats(w http.ResponseWriter, r *http.Request) {
	data := getBaseData()
	data.Data = metrics.GetFormatted()
	tmplCache["stats.html"].ExecuteTemplate(w, "stats.html", data)
}

// ServeSettings manages application configuration changes
func ServeSettings(w http.ResponseWriter, r *http.Request) {
	data := getBaseData()
	cfg := config.Get()

	if r.Method == http.MethodPost {
		port := r.FormValue("port")
		theme := r.FormValue("theme")
		locale := r.FormValue("locale")
		password := r.FormValue("password")

		if port != "" {
			cfg.SetPort(port)
		}
		if theme == "dark" || theme == "light" {
			cfg.SetTheme(theme)
		}
		if locale == "en" || locale == "tr" {
			cfg.SetLocale(locale)
		}
		if password != "" {
			cfg.SetPassword(password)
		}

		// Process Notification Settings if present in the form payload
		r.ParseForm()
		if _, ok := r.PostForm["tg_bot_token"]; ok {
			notif := cfg.GetNotif()
			notif.TgBotToken = r.FormValue("tg_bot_token")
			notif.TgChatId = r.FormValue("tg_chat_id")
			notif.WebhookUrl = r.FormValue("webhook_url")
			notif.SmtpHost = r.FormValue("smtp_host")
			notif.SmtpPort = r.FormValue("smtp_port")
			notif.SmtpUser = r.FormValue("smtp_user")
			notif.SmtpPass = r.FormValue("smtp_pass")
			notif.SmtpTo = r.FormValue("smtp_to")
			cfg.SetNotif(notif)
		}

		// Save everything to .env physically
		cfg.SaveEnv()

		data = getBaseData() // Refresh references
		data.Info = string(data.T("SettingsSaved"))

		// Note: Restarting the HTTP server to bind a new port natively is complex without a manager.
		// In a containerized context, a process restart is often preferred. 
		// We could use a graceful restart package or signal here if needed.
		// For now, settings are saved in memory and will persist until container restarts.
		if port != "" && port != cfg.GetPort() {
			go func() {
				// We won't actually kill the process directly, just warning the user
				// A real production system might listen for SIGHUP or similar.
			}()
		}
	}

	// Prepare current configuration to show in inputs
	currentConfig := struct {
		Port     string
		Theme    string
		Locale   string
		Notif    config.NotificationConfig
	}{
		Port:     cfg.GetPort(),
		Theme:    cfg.GetTheme(),
		Locale:   cfg.GetLocale(),
		Notif:    cfg.GetNotif(),
	}
	
	data.Data = currentConfig
	tmplCache["settings.html"].ExecuteTemplate(w, "base.html", data)
}

// ServeAutomation renders the rules building interface
func ServeAutomation(w http.ResponseWriter, r *http.Request) {
	data := getBaseData()
	// Get rules from config state
	data.Data = config.Get().GetRules()
	
	// Check for ?info= query params for banner
	if info := r.URL.Query().Get("info"); info != "" {
		data.Info = info
	}

	tmplCache["automation.html"].ExecuteTemplate(w, "base.html", data)
}

// AddAutomationRule processes form submissions and adds to state
func AddAutomationRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/automation", http.StatusFound)
		return
	}

	cfg := config.Get()
	rules := cfg.GetRules()

	threshold, _ := strconv.ParseFloat(r.FormValue("threshold"), 64)
	duration, _ := strconv.Atoi(r.FormValue("duration"))
	cooldown, _ := strconv.Atoi(r.FormValue("cooldown"))

	newRule := config.AlertRule{
		ID:                  fmt.Sprintf("%d", time.Now().UnixNano()),
		MetricType:          r.FormValue("metric"),
		Operator:            r.FormValue("operator"),
		ThresholdPercent:    threshold,
		DurationSeconds:     duration,
		CooldownSeconds:     cooldown,
		SentCount:           0,
		MessageTemplate:     r.FormValue("message_template"),
		ShellCommand:        r.FormValue("command"),
		NotificationChannel: r.FormValue("channel"),
		IsActive:            true,
	}

	rules = append(rules, newRule)
	cfg.SetRules(rules)

	http.Redirect(w, r, "/automation?info=Rule+Successfully+Added", http.StatusFound)
}

// ToggleAutomationRule processes a request to flip IsActive bool on a rule
func ToggleAutomationRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	id := r.FormValue("id")
	cfg := config.Get()
	rules := cfg.GetRules()

	for i, rule := range rules {
		if rule.ID == id {
			rules[i].IsActive = !rules[i].IsActive
			// Reset tracking logic when toggling
			rules[i].ViolatingSince = nil
			rules[i].HasTriggered = false
			break
		}
	}
	cfg.SetRules(rules)
	http.Redirect(w, r, "/automation?info=Rule+Status+Updated", http.StatusFound)
}

// DeleteAutomationRule processes removing a rule entirely
func DeleteAutomationRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	id := r.FormValue("id")
	cfg := config.Get()
	rules := cfg.GetRules()

	var newRules []config.AlertRule
	for _, rule := range rules {
		if rule.ID != id {
			newRules = append(newRules, rule)
		}
	}
	cfg.SetRules(newRules)
	http.Redirect(w, r, "/automation?info=Rule+Deleted", http.StatusFound)
}

// TestNotification handles manual channel tests from the Settings page
func TestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	channel := r.FormValue("channel")
	alerting.SendTestNotification(channel)
	
	// HTMX will swap the button with this success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<span class="text-green-600 dark:text-green-400 font-semibold text-sm">Test Sent! 🚀</span>`))
}
