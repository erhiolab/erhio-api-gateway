package email

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"errors"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// MailPayload 邮件发送负载
type MailPayload struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	HTML    bool     `json:"html"`
}

// SendMail 发送邮件
func SendMail(payload *MailPayload) error {
	cfg := config.Get().DatabaseConfig.Email
	retry := cfg.MaxRetry
	if retry <= 0 {
		retry = 3
	}
	var err error
	for i := 0; i <= retry; i++ {
		err = sendWithTimeout(func() error {
			return doSend(payload)
		}, time.Duration(cfg.Timeout)*time.Second)
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	logger.Log.Error("发送邮件失败", zap.Error(err))
	return err
}

// doSend 发送邮件
func doSend(payload *MailPayload) error {
	cfg := config.Get().DatabaseConfig.Email
	mail := gomail.NewMessage()
	mail.SetHeader("From", cfg.Username)
	mail.SetHeader("To", payload.To...)
	mail.SetHeader("Subject", payload.Subject)
	if payload.HTML {
		mail.SetBody("text/html", payload.Body)
	} else {
		mail.SetBody("text/plain", payload.Body)
	}
	dialer := gomail.NewDialer(
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
	)
	return dialer.DialAndSend(mail)
}

// sendWithTimeout 发送邮件并设置超时
func sendWithTimeout(fn func() error, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		logger.Log.Error("发送邮件超时", zap.Duration("timeout", timeout))
		return errors.New("发送邮件超时")
	}
}
