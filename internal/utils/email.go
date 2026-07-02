package utils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// EmailConfig holds SMTP configuration read from environment variables.
//
//	SMTP_HOST  — e.g. smtp.gmail.com
//	SMTP_PORT  — e.g. 587 (STARTTLS) or 465 (SSL) [default: 587]
//	SMTP_USER  — sender login / username
//	SMTP_PASS  — sender password or app-password
//	SMTP_FROM  — From address shown to recipients (defaults to SMTP_USER)
type EmailConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func loadEmailConfig() *EmailConfig {
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}
	port := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	return &EmailConfig{Host: host, Port: port, User: user, Pass: pass, From: from}
}

// SendEmail sends a plain-text email. It silently returns nil when SMTP_HOST
// is not configured (so callers don't need to guard every call site).
func SendEmail(to, subject, body string) error {
	cfg := loadEmailConfig()
	if cfg.Host == "" || cfg.User == "" {
		return nil // SMTP not configured — skip silently
	}

	msg := buildMessage(cfg.From, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.Port == 465 {
		return sendSSL(cfg, addr, to, msg)
	}
	return sendSTARTTLS(cfg, addr, to, msg)
}

// sendSTARTTLS uses port 587 with STARTTLS upgrade.
func sendSTARTTLS(cfg *EmailConfig, addr, to string, msg []byte) error {
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
}

// sendSSL uses port 465 with an immediate TLS connection.
func sendSSL(cfg *EmailConfig, addr, to string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp ssl dial: %w", err)
	}
	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp ssl client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp ssl auth: %w", err)
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("smtp ssl mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp ssl rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp ssl data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp ssl write: %w", err)
	}
	return w.Close()
}

func buildMessage(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
