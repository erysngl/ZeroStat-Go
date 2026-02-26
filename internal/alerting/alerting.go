package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/erysngl/zerostat/internal/config"
	"github.com/erysngl/zerostat/internal/metrics"
)

// Start Engine kicks off the stateful background evaluator
func StartEngine() {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			evaluateRules()
		}
	}()
}

func evaluateRules() {
	cfg := config.Get()
	rules := cfg.GetRules()
	
	if len(rules) == 0 {
		return
	}

	stats := metrics.GetStats()

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		currentValue := getMetricValue(rule.MetricType, stats)
		
		isViolating := false
		switch rule.Operator {
		case ">":
			isViolating = currentValue > rule.ThresholdPercent
		case "<":
			isViolating = currentValue < rule.ThresholdPercent
		case "==":
			// Floating point exact match is tricky, let's use a very small epsilon
			isViolating = math.Abs(currentValue-rule.ThresholdPercent) < 0.01
		default:
			// Fallback to strict greater equals if undefined
			isViolating = currentValue >= rule.ThresholdPercent
		}

		if isViolating {
			if rule.ViolatingSince == nil {
				// Record First Violation Time
				now := time.Now()
				cfg.UpdateRuleState(rule.ID, &now, false)
				continue
			}

			// Debounce check against Seconds
			violationDuration := time.Since(*rule.ViolatingSince)
			requiredDuration := time.Duration(rule.DurationSeconds) * time.Second

			if violationDuration >= requiredDuration {
				if !rule.HasTriggered {
					// Trigger the first action
					executeAction(rule, currentValue)
					cfg.UpdateRuleState(rule.ID, rule.ViolatingSince, true)
					cfg.MarkRuleSent(rule.ID)
				} else if rule.CooldownSeconds > 0 && rule.LastSentAt != nil {
					// Check if Cooldown elapsed for subsequent alerts
					if time.Since(*rule.LastSentAt) >= time.Duration(rule.CooldownSeconds)*time.Second {
						executeAction(rule, currentValue)
						cfg.MarkRuleSent(rule.ID)
					}
				}
			}
		} else {
			// Did it recover?
			if rule.HasTriggered {
				sendRecoveryNotification(rule, currentValue)
			}
			// Reset state immediately since it dropped
			if rule.ViolatingSince != nil || rule.HasTriggered {
				cfg.UpdateRuleState(rule.ID, nil, false)
			}
		}
	}
}

func getMetricValue(metricType string, stats *metrics.SystemStats) float64 {
	switch metricType {
	case "CPU":
		return stats.CPUUsage
	case "RAM":
		return stats.MemUsage
	case "Disk":
		return stats.DiskUsage
	}
	return 0.0
}

func executeAction(rule config.AlertRule, currentVal float64) {
	log.Printf("[ALERT] Rule triggered! %s %s %.2f%% (Current: %.2f%%). Action: %s", 
		rule.MetricType, rule.Operator, rule.ThresholdPercent, currentVal, rule.ShellCommand)
	
	// Send notification if requested
	if rule.NotificationChannel != "" && rule.NotificationChannel != "none" {
		msg := buildMessage(rule, currentVal, false)
		go sendNotification(rule.NotificationChannel, msg)
	}

	if rule.ShellCommand != "" {
		go executeSafeShell(rule.ShellCommand)
	}
}

func containsShellInjection(cmd string) bool {
	// Let's create a strict whitelist of commands the UI generates via templates
	whitelist := []string{
		"docker stop $(docker ps -q)",
		"sync; echo 1 > /proc/sys/vm/drop_caches",
		"systemctl restart my-app",
	}
	
	for _, w := range whitelist {
		if strings.TrimSpace(cmd) == w {
			return false // Safe
		}
	}

	// Blacklist dangerous metacharacters
	dangerous := []string{";", "&", "|", "$", ">", "<", "`", "\n"}
	for _, char := range dangerous {
		if strings.Contains(cmd, char) {
			return true
		}
	}
	return false
}

func executeSafeShell(command string) {
	if containsShellInjection(command) {
		log.Printf("[ALERT-SECURITY] Blocked potentially unsafe shell command: %s", command)
		return
	}

	// Execute the command safely, killing it if it takes longer than 30s
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[ALERT-ACTION] Executing shell command: %s", command)
	// Sh -c format supports standard bash pipes and operators like $(docker ps -q)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("[ALERT-ACTION] Command timed out after 30s: %s", command)
		return
	}
	
	if err != nil {
		log.Printf("[ALERT-ACTION] Execution failed: %v | Output: %s", err, string(output))
		return
	}
	
	log.Printf("[ALERT-ACTION] Execution succeeded. Output: %s", string(output))
}

func sendRecoveryNotification(rule config.AlertRule, currentVal float64) {
	log.Printf("[RECOVERY] System recovered for %s rule. Current Value: %.2f%%.", rule.MetricType, currentVal)
	
	if rule.NotificationChannel != "" && rule.NotificationChannel != "none" {
		msg := buildMessage(rule, currentVal, true)
		go sendNotification(rule.NotificationChannel, msg)
	}
}

func buildMessage(rule config.AlertRule, currentVal float64, isRecovery bool) string {
	template := rule.MessageTemplate
	if isRecovery && template == "" {
		template = "[ZeroStat-Go] {hostname} Recovery: {metric} is now at {value}%. System is safe."
	} else if template == "" {
		template = "[ZeroStat-Go] {hostname} Warning: {metric} value is {value}%! (Threshold: {operator}{threshold}, Duration: {duration}s)"
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-host"
	}

	msg := strings.ReplaceAll(template, "{hostname}", hostname)
	msg = strings.ReplaceAll(msg, "{metric}", rule.MetricType)
	msg = strings.ReplaceAll(msg, "{value}", fmt.Sprintf("%.2f", currentVal))
	msg = strings.ReplaceAll(msg, "{threshold}", fmt.Sprintf("%.2f", rule.ThresholdPercent))
	msg = strings.ReplaceAll(msg, "{operator}", rule.Operator)
	msg = strings.ReplaceAll(msg, "{duration}", fmt.Sprintf("%d", rule.DurationSeconds))

	return msg
}

func sendNotification(channel, message string) {
	log.Printf("[NOTIFICATION-DISPATCH] Channel: %s | Payload: %s", channel, message)
	
	notif := config.Get().GetNotif()
	
	switch channel {
	case "webhook":
		if notif.WebhookUrl != "" {
			payload := map[string]string{"text": message}
			jsonPayload, _ := json.Marshal(payload)
			resp, err := http.Post(notif.WebhookUrl, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Printf("[ERROR] Webhook failed: %v", err)
			} else {
				resp.Body.Close()
			}
		} else {
			log.Println("[WARNING] Webhook URL not configured.")
		}
	case "telegram":
		if notif.TgBotToken != "" && notif.TgChatId != "" {
			url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", notif.TgBotToken)
			payload := map[string]string{"chat_id": notif.TgChatId, "text": message}
			jsonPayload, _ := json.Marshal(payload)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Printf("[ERROR] Telegram HTTP failed: %v", err)
			} else {
				resp.Body.Close()
			}
		} else {
			log.Println("[WARNING] Telegram target not configured.")
		}
	case "email":
		if notif.SmtpHost != "" && notif.SmtpTo != "" {
			auth := smtp.PlainAuth("", notif.SmtpUser, notif.SmtpPass, notif.SmtpHost)
			addr := fmt.Sprintf("%s:%s", notif.SmtpHost, notif.SmtpPort)
			msg := []byte("To: " + notif.SmtpTo + "\r\n" +
				"Subject: ZeroStat-Go Alert\r\n" +
				"\r\n" + message + "\r\n")
			err := smtp.SendMail(addr, auth, notif.SmtpUser, []string{notif.SmtpTo}, msg)
			if err != nil {
				log.Printf("[ERROR] Email SMTP failed: %v", err)
			}
		} else {
			log.Println("[WARNING] SMTP settings missing.")
		}
	}
}

// SendTestNotification exports the internal dispatcher for manual testing from UI
func SendTestNotification(channel string) {
	sendNotification(channel, "ZeroStat-Go Test Message - System Successfully Verified! 🚀")
}
