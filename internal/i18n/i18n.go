package i18n

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

type Dictionary map[string]string

var dictionaries = map[string]Dictionary{}

var fallbackEn = map[string]string{
	"LoginTitle":     "Secure Login",
	"Password":       "Password",
	"LoginBtn":       "Sign In",
	"InvalidCreds":   "Invalid credentials",
	"Dashboard":      "Dashboard",
	"Settings":       "Settings",
	"Logout":         "Logout",
	"CPUUsage":       "CPU Usage",
	"RAMUsage":       "RAM Usage",
	"DiskUsage":      "Disk Capacity",
	"NetworkRx":      "Network (Rx)",
	"NetworkTx":      "Network (Tx)",
	"Configuration":  "Configuration",
	"AppPort":        "Application Port",
	"AppTheme":       "Theme",
	"AppLocale":      "Language",
	"AdminPassword":  "Admin Password",
	"SaveSettings":   "Save Settings",
	"ThemeDark":      "Dark",
	"ThemeLight":     "Light",
	"LocaleEn":       "English",
	"LocaleTr":       "Türkçe",
	"SettingsSaved":  "Settings saved successfully",
	"Automation":         "Automation",
	"GuardrailsDesc":     "Guardrails & Safety Actions: Define rules that trigger automatically when a metric breaches the threshold for the specified duration.",
	"CreateRule":         "Create New Metric Rule",
	"TargetMetric":       "Target Metric",
	"ThresholdPct":       "Threshold (%)",
	"DebounceSec":        "Debounce Duration (Sec)",
	"DebounceHint":       "Must sustain breach for this many seconds",
	"ShellCmd":           "Execute Shell Command",
	"Optional":           "Optional",
	"SelectPreset":       "-- Select a safe preset template --",
	"PresetDockerStop":   "Halt all running Docker containers",
	"PresetClearCache":   "Clear OS PageCache",
	"PresetRestartApp":   "Restart your main application service",
	"PlaceholderCmd":     "e.g. docker stop $(docker ps -q)",
	"NotifChannel":       "Notification Channel",
	"Disabled":           "Disabled",
	"Webhook":            "Webhook (POST)",
	"TelegramBot":        "Telegram Bot",
	"EmailSMTP":          "Email (SMTP)",
	"AddRule":            "Add Rule",
	"ActiveRulesEngine":  "Active Rules Engine",
	"NoRulesConfigured":  "No automation rules configured yet.",
	"DebounceLabel":      "Debounce:",
	"ViolationActive":    "VIOLATION ACTIVE",
	"ChannelLabel":       "Channel:",
	"Disable":            "Disable",
	"Enable":             "Enable",
	"Remove":             "Remove",
    "NotificationConfig": "Notification Configuration",
    "TgBotToken":         "Telegram Bot Token",
    "TgChatId":           "Telegram Chat ID",
    "WebhookUrl":         "Webhook URL (POST)",
    "SmtpHost":           "SMTP Host",
    "SmtpPort":           "SMTP Port",
    "SmtpUser":           "SMTP User",
    "SmtpPass":           "SMTP Password",
    "SmtpTo":             "Recipient Email",
    "OperatorLabel":      "Logic Operator",
    "OpGreater":          "Greater Than (>)",
    "OpLess":             "Less Than (<)",
    "OpEqual":            "Equals (==)",
    "MessageTemplate":    "Custom Message Template",
    "TemplateHint":       "Available tags: {hostname}, {metric}, {value}, {threshold}, {operator}, {duration}",
    "TemplateCritical":   "Critical Load!",
    "TemplateWarning":    "Soft Warning",
    "TemplateRecovery":   "Recovery",
    "CooldownSec":        "Cooldown (Sec)",
    "CooldownHint":       "Wait this long before sending another alert",
    "TagLegendTitle":     "Tag Legend",
    "TagHostname":        "Server Hostname",
    "TagMetric":          "Monitored Metric",
    "TagValue":           "Current Actual Value",
    "TagThreshold":       "Set Limit Value",
    "TagOperator":        "Logic Operator (>,<,==)",
    "TagDuration":        "Debounce Seconds",
    "SentCountLabel":     "Total Sent",
    "CooldownLabel":      "Cooldown:",
}

func Init() {
	loadLocale("en")
	loadLocale("tr")
}

func loadLocale(lang string) {
	// Look for the locale file
	path := filepath.Join("locales", lang+".json")
	file, err := os.ReadFile(path)
	if err != nil {
		log.Printf("i18n warning: could not load %s locale file: %v\n", lang, err)
		return
	}

	var dict Dictionary
	if err := json.Unmarshal(file, &dict); err != nil {
		log.Printf("i18n warning: could not parse %s locale file: %v\n", lang, err)
		return
	}

	dictionaries[lang] = dict
}

// T translates a key based on the provided locale. Falls back to English if missing.
func T(locale, key string) string {
	dict, ok := dictionaries[locale]
	if !ok {
		dict = dictionaries["en"]
	}

	val, exists := dict[key]
	if !exists {
		// Ultimate Fallback: return hardcoded default
		if fallbackVal, ok := fallbackEn[key]; ok {
			return fallbackVal
		}
		// If utterly missing, return the Key string itself to avoid blanks
		return key
	}
	return val
}

// TFunc is a helper to inject the translation function into HTML templates.
func TFunc(locale string) func(string) template.HTML {
	return func(key string) template.HTML {
		return template.HTML(T(locale, key))
	}
}
