package config

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type NotificationConfig struct {
	TgBotToken string
	TgChatId   string
	WebhookUrl string
	SmtpHost   string
	SmtpPort   string
	SmtpUser   string
	SmtpPass   string
	SmtpTo     string
}

type Config struct {
	mu         sync.RWMutex
	Port       string
	Password   string
	Theme      string
	Locale     string
	AlertRules []AlertRule
	Notif      NotificationConfig
}

type AlertRule struct {
	ID                  string
	MetricType          string  // CPU, RAM, Disk
	Operator            string  // e.g., ">", "<", "="
	ThresholdPercent    float64 // 90.0, etc
	DurationSeconds     int     // 600 (10 minutes)
	CooldownSeconds     int     // Wait time before triggering again
	SentCount           int     // Number of times triggered
	MessageTemplate     string  // e.g., "CPU usage is {{.Value}}%, exceeding {{.Threshold}}%"
	ShellCommand        string  // e.g. docker stop $(docker ps -q)
	NotificationChannel string  // webhook, telegram, etc
	IsActive            bool
	
	// Internal State
	ViolatingSince *time.Time
	HasTriggered   bool
	LastSentAt     *time.Time
}

var (
	appConfig *Config
	once      sync.Once
)

// Init loads initial configuration from .env or defaults
func Init() {
	once.Do(func() {
		_ = godotenv.Load() // Ignore error if .env doesn't exist

		port := os.Getenv("ZEROSTAT_PORT")
		if port == "" {
			port = "9124"
		}

		password := os.Getenv("ZEROSTAT_PASSWORD")
		if password == "" {
			password = "admin" // Default for local testing
		}

		notif := NotificationConfig{
			TgBotToken: os.Getenv("TG_BOT_TOKEN"),
			TgChatId:   os.Getenv("TG_CHAT_ID"),
			WebhookUrl: os.Getenv("WEBHOOK_URL"),
			SmtpHost:   os.Getenv("SMTP_HOST"),
			SmtpPort:   os.Getenv("SMTP_PORT"),
			SmtpUser:   os.Getenv("SMTP_USER"),
			SmtpPass:   os.Getenv("SMTP_PASS"),
			SmtpTo:     os.Getenv("SMTP_TO"),
		}

		appConfig = &Config{
			Port:       port,
			Password:   password,
			Theme:      "dark", // default theme
			Locale:     "en",   // default locale
			AlertRules: make([]AlertRule, 0),
			Notif:      notif,
		}
	})
}

// Get access the singleton configuration
func Get() *Config {
	if appConfig == nil {
		log.Fatal("Config not initialized")
	}
	return appConfig
}

func (c *Config) GetPort() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Port
}

func (c *Config) SetPort(port string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Port = port
}

func (c *Config) GetPassword() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Password
}

func (c *Config) SetPassword(pwd string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Password = pwd
}

func (c *Config) GetTheme() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Theme
}

func (c *Config) SetTheme(theme string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Theme = theme
}

func (c *Config) GetLocale() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Locale
}

func (c *Config) SetLocale(locale string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Locale = locale
}

func (c *Config) GetRules() []AlertRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to avoid race conditions when the alerting engine reads
	rules := make([]AlertRule, len(c.AlertRules))
	copy(rules, c.AlertRules)
	return rules
}

func (c *Config) SetRules(rules []AlertRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AlertRules = rules
}

func (c *Config) UpdateRuleState(id string, violatingSince *time.Time, hasTriggered bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, r := range c.AlertRules {
		if r.ID == id {
			c.AlertRules[i].ViolatingSince = violatingSince
			c.AlertRules[i].HasTriggered = hasTriggered
			break
		}
	}
}

func (c *Config) MarkRuleSent(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, r := range c.AlertRules {
		if r.ID == id {
			c.AlertRules[i].SentCount++
			now := time.Now()
			c.AlertRules[i].LastSentAt = &now
			break
		}
	}
}

func (c *Config) GetNotif() NotificationConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Notif
}

func (c *Config) SetNotif(n NotificationConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Notif = n
}

func (c *Config) SaveEnv() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	envMap := map[string]string{
		"ZEROSTAT_PORT":     c.Port,
		"ZEROSTAT_PASSWORD": c.Password,
		"TG_BOT_TOKEN":      c.Notif.TgBotToken,
		"TG_CHAT_ID":        c.Notif.TgChatId,
		"WEBHOOK_URL":       c.Notif.WebhookUrl,
		"SMTP_HOST":         c.Notif.SmtpHost,
		"SMTP_PORT":         c.Notif.SmtpPort,
		"SMTP_USER":         c.Notif.SmtpUser,
		"SMTP_PASS":         c.Notif.SmtpPass,
		"SMTP_TO":           c.Notif.SmtpTo,
	}

	godotenv.Write(envMap, ".env")
	os.Chmod(".env", 0600)
}
