package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSender 基于 stdlib net/smtp 的邮件发送器
// 支持 TLS/Plain 两种连接方式，通过 goroutine+select 实现 context 超时控制
type SMTPSender struct {
	host     string // SMTP 服务器地址，如 smtp.qq.com
	port     string // SMTP 端口号
	username string // SMTP 登录账号
	password string // SMTP 登录密码或授权码
	from     string // 发件人显示地址，如 "CloudEmu <noreply@example.com>"
	useTLS   bool   // 是否使用 TLS 连接
}

// NewSMTPSender 创建 SMTP 邮件发送器，port 为端口号（如 587），from 为发件人显示地址
func NewSMTPSender(host string, port int, username, password, from string, useTLS bool) *SMTPSender {
	return &SMTPSender{
		host: host, port: fmt.Sprintf("%d", port),
		username: username, password: password, from: from, useTLS: useTLS,
	}
}

// Send 发送邮件，通过 goroutine 包裹实际发送逻辑以支持 context 超时（10秒）
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg := buildMessage(s.from, to, subject, body)

	errCh := make(chan error, 1)
	go func() {
		if s.useTLS {
			errCh <- s.sendWithTLS(to, msg)
		} else {
			errCh <- s.sendPlain(to, msg)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-sendCtx.Done():
		return fmt.Errorf("smtp send timeout: %w", sendCtx.Err())
	}
}

// sendPlain 使用普通 SMTP 连接（无 TLS）
func (s *SMTPSender) sendPlain(to string, msg []byte) error {
	addr := net.JoinHostPort(s.host, s.port)
	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
	}
	return smtp.SendMail(addr, nil, s.from, []string{to}, msg)
}

// sendWithTLS 使用 TLS 加密连接发送邮件
func (s *SMTPSender) sendWithTLS(to string, msg []byte) error {
	addr := net.JoinHostPort(s.host, s.port)
	tlsConfig := &tls.Config{ServerName: s.host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Quit()

	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

// NoopSender 空实现发送器，SMTP 未配置时使用
// 仅通过 slog.Debug 记录邮件内容，方便开发环境调试
type NoopSender struct{}

func (s *NoopSender) Send(ctx context.Context, to, subject, body string) error {
	slog.Debug("email not sent (no SMTP configured)",
		"to", to,
		"subject", subject,
		"body", body,
	)
	return nil
}

// buildMessage 构建符合 RFC 2822 的邮件消息
// 使用 text/plain + UTF-8 编码，支持中文邮件正文
func buildMessage(from, to, subject, body string) []byte {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
