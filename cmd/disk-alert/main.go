package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path := envDefault("DISK_PATH", "/")
	threshold, err := strconv.ParseFloat(envDefault("DISK_THRESHOLD_PERCENT", "80"), 64)
	if err != nil || threshold <= 0 || threshold >= 100 {
		return errors.New("DISK_THRESHOLD_PERCENT must be between 0 and 100")
	}
	used, err := diskUsedPercent(path)
	if err != nil {
		return err
	}
	statePath := envDefault("DISK_ALERT_STATE_FILE", "/var/lib/linka-metric/disk-alert.state")
	alerted, err := readAlertState(statePath)
	if err != nil {
		return err
	}
	if used < threshold {
		if alerted {
			return writeAlertState(statePath, false)
		}
		return nil
	}
	if alerted {
		return nil
	}
	message := fmt.Sprintf("Использование диска %s достигло %.1f%% (порог %.1f%%).", path, used, threshold)
	if err := sendMail(
		os.Getenv("SMTP_ADDR"),
		os.Getenv("SMTP_FROM"),
		envDefault("SMTP_TO", "ivan@aacidov.ru"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
		message,
	); err != nil {
		return err
	}
	return writeAlertState(statePath, true)
}

func diskUsedPercent(path string) (float64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, fmt.Errorf("stat filesystem: %w", err)
	}
	if stats.Blocks == 0 {
		return 0, errors.New("filesystem reports zero blocks")
	}
	return (1 - float64(stats.Bavail)/float64(stats.Blocks)) * 100, nil
}

func readAlertState(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read alert state: %w", err)
	}
	return strings.TrimSpace(string(data)) == "alerted", nil
}

func writeAlertState(path string, alerted bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create alert state directory: %w", err)
	}
	state := "ok\n"
	if alerted {
		state = "alerted\n"
	}
	if err := os.WriteFile(path, []byte(state), 0o600); err != nil {
		return fmt.Errorf("write alert state: %w", err)
	}
	return nil
}

func sendMail(address, from, to, username, password, body string) error {
	if address == "" || from == "" {
		return errors.New("SMTP_ADDR and SMTP_FROM are required when an alert is sent")
	}
	if strings.ContainsAny(from+to, "\r\n") {
		return errors.New("invalid email address")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("parse SMTP_ADDR: %w", err)
	}
	connection, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect SMTP: %w", err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(30 * time.Second))
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return fmt.Errorf("start SMTP client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("start SMTP TLS: %w", err)
	}
	if username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return fmt.Errorf("authenticate SMTP: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	subject := mime.QEncoding.Encode("utf-8", "LINKa Plays Metric: заполнение диска")
	message := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body + "\r\n"
	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	return client.Quit()
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
