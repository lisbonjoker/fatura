package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
)

func sendInvoiceEmail(to, subject, pdfPath string, cfg SMTPConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("SMTP não configurado — adicione as credenciais em ~/.invoice/config.yaml")
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return fmt.Errorf("não foi possível ler o PDF: %w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Plain-text body
	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=utf-8")
	tw, _ := w.CreatePart(th)
	fmt.Fprintf(tw, "Exmo(a) Cliente,\n\nEm anexo encontra a fatura solicitada.\n\nCom os melhores cumprimentos.")

	// PDF attachment
	ah := make(textproto.MIMEHeader)
	ah.Set("Content-Type", "application/pdf")
	ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(pdfPath)))
	ah.Set("Content-Transfer-Encoding", "base64")
	aw, _ := w.CreatePart(ah)
	fmt.Fprint(aw, base64.StdEncoding.EncodeToString(pdfData))
	w.Close()

	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n",
		from, to, subject, w.Boundary(),
	)
	msg := append([]byte(header), buf.Bytes()...)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	// Use STARTTLS when port is 587, plain TLS when 465.
	if cfg.Port == 465 {
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("erro de ligação TLS: %w", err)
		}
		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Auth(auth); err != nil {
			return err
		}
		if err := c.Mail(from); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		wc, err := c.Data()
		if err != nil {
			return err
		}
		_, err = wc.Write(msg)
		wc.Close()
		return err
	}

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
