package emailing

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

type SMTPTransport struct{}

func (SMTPTransport) Send(config SMTPSettings, password string, message Message) error {
	address := net.JoinHostPort(config.Host, fmt.Sprint(config.Port))
	var client *smtp.Client
	var err error
	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
	if config.Security == "TLS" {
		conn, e := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", address, tlsConfig)
		if e != nil {
			return e
		}
		client, err = smtp.NewClient(conn, config.Host)
	} else {
		conn, e := net.DialTimeout("tcp", address, 15*time.Second)
		if e != nil {
			return e
		}
		client, err = smtp.NewClient(conn, config.Host)
		if err == nil && config.Security == "STARTTLS" {
			err = client.StartTLS(tlsConfig)
		}
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if config.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", config.Username, password, config.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(config.FromEmail); err != nil {
		return err
	}
	if err = client.Rcpt(message.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	payload, err := mimeMessage(config, message)
	if err != nil {
		return err
	}
	if _, err = writer.Write(payload); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func mimeMessage(config SMTPSettings, message Message) ([]byte, error) {
	var out bytes.Buffer
	mixed := multipart.NewWriter(&out)
	fmt.Fprintf(&out, "From: %s <%s>\r\n", sanitizeHeader(config.FromName), config.FromEmail)
	fmt.Fprintf(&out, "To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", message.To, sanitizeHeader(message.Subject), mixed.Boundary())
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := mixed.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}
	qp := quotedprintable.NewWriter(part)
	if _, err = qp.Write([]byte(message.HTML)); err != nil {
		return nil, err
	}
	if err = qp.Close(); err != nil {
		return nil, err
	}
	for _, attachment := range message.Attachments {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", attachment.ContentType)
		header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(attachment.Filename, `"`, "")))
		header.Set("Content-Transfer-Encoding", "base64")
		p, e := mixed.CreatePart(header)
		if e != nil {
			return nil, e
		}
		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		for len(encoded) > 76 {
			fmt.Fprintf(p, "%s\r\n", encoded[:76])
			encoded = encoded[76:]
		}
		fmt.Fprintf(p, "%s\r\n", encoded)
	}
	if err = mixed.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
