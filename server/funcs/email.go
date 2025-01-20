package funcs

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
	"server/config"
	"server/models"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/mailgun/mailgun-go/v4"
)

// CheckEmail checks to see if the email is valid
func CheckEmail(email string) bool {
	_, err := mail.ParseEmail(email)
	return err == nil
}

// SendJudgeEmail sends an email to the judge with their code
func SendJudgeEmail(judge *models.Judge, hostname string) error {
	// If mailgun API key exists, send email with mailgun
	mailgunApiKey := config.GetOptEnv("MAILGUN_API_KEY", "")
	mailgunDomain := config.GetOptEnv("MAILGUN_DOMAIN", "")
	mailgunIsEu := config.GetOptEnv("MAILGUN_IS_EU_TRUE_FALSE", "false")
	if mailgunApiKey != "" && mailgunDomain != "" {
		return mailgunEmail(mailgunApiKey, mailgunDomain, mailgunIsEu, judge, hostname)
	}

	// If sendgrid API key exists, send email with sendgrid
	sendgridApiKey := config.GetOptEnv("SENDGRID_API_KEY", "")
	if sendgridApiKey != "" {
		return sendgridEmail(sendgridApiKey, judge, hostname)
	}

	// Sender info
	from := config.GetEnv("EMAIL_FROM")
	username := config.GetEnv("EMAIL_USERNAME")
	password := config.GetEnv("EMAIL_PASSWORD")

	// Receiver info
	to := []string{
		judge.Email,
	}

	// SMTP server configuration
	smtpHost := config.GetEnv("EMAIL_HOST")
	smtpPort := config.GetOptEnv("EMAIL_PORT", "587")

	// Auth
	auth := smtp.PlainAuth("jury", username, password, smtpHost)

	// Other info
	appName := config.GetEnv("VITE_JURY_NAME")

	// Message body
	var body bytes.Buffer
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: Jury Judging Platform [%s] \n%s\n\n", appName, mimeHeaders)))

	// Fill Template
	html, err := FillTemplate(judge.Name, hostname, judge.Code, appName)
	if err != nil {
		return err
	}
	body.Write(html)

	// Send email!
	err = sendEmailWithTimeout(smtpHost, smtpPort, auth, from, to, body.Bytes())
	return err
}

// FillTemplate fills the email template with the given data
func FillTemplate(name string, baseUrl string, code string, appName string) ([]byte, error) {
	var body bytes.Buffer

	t, err := template.ParseFiles("email.html")
	if err != nil {
		return nil, err
	}

	t.Execute(&body, struct {
		Name      string
		BaseUrl   string
		CodeLeft  string
		CodeRight string
		Code      string
		AppName   string
	}{
		Name:      name,
		BaseUrl:   baseUrl,
		CodeLeft:  code[:4],
		CodeRight: code[4:],
		Code:      code,
		AppName:   appName,
	})

	return body.Bytes(), nil
}

// Send email with Sendgrid
func sendgridEmail(sendgridApiKey string, judge *models.Judge, hostname string) error {
	appName := config.GetEnv("VITE_JURY_NAME")

	from := mail.NewEmail(config.GetEnv("EMAIL_FROM_NAME"), config.GetEnv("EMAIL_FROM"))
	subject := "Jury Judging Platform [" + appName + "]"
	to := mail.NewEmail(judge.Name, judge.Email)

	htmlContent, err := FillTemplate(judge.Name, hostname, judge.Code, appName)
	if err != nil {
		return err
	}

	message := mail.NewSingleEmail(from, subject, to, "", string(htmlContent))
	client := sendgrid.NewSendClient(sendgridApiKey)
	_, err = client.Send(message)
	return err
}

// Send email with Mailgun
func mailgunEmail(mailgunApiKey string, mailgunDomain string, mailgunIsEu string, judge *models.Judge, hostname string) error {
	// taken from https://documentation.mailgun.com/docs/mailgun/sdk/go_sdk/
	appName := config.GetEnv("VITE_JURY_NAME")

	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(mailgunDomain, mailgunApiKey)
	
	//When you have an EU-domain, you must specify the endpoint:
	if strings.ToLower(mailgunIsEu) == "true" {
		mg.SetAPIBase("https://api.eu.mailgun.net/v3")
	}


	from := formatEmailWithName(config.GetEnv("EMAIL_FROM_NAME"), config.GetEnv("EMAIL_FROM"))
	subject := "Jury Judging Platform [" + appName + "]"
	to := formatEmailWithName(judge.Name, judge.Email)

	htmlContent, err := FillTemplate(judge.Name, hostname, judge.Code, appName)
	if err != nil {
		return err
	}

	//The message object allows you to add attachments and Bcc recipients
	message := mailgun.NewMessage(from, subject, "", to)
	message.SetHTML(string(htmlContent))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10-second timeout
	_, _, err = mg.Send(ctx, message)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("ID: %s Resp: %s\n", id, resp)

	return err
}

func sendEmailWithTimeout(host string, port string, auth smtp.Auth, from string, to []string, body []byte) error {
	// Dial SMTP with 5 second timeout
	conn, err := net.DialTimeout("tcp", host+":"+port, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to email server: %v", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()

	// Initiate TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Set to true for self-signed certificates, but ideally use false and verify the server's certificate
		ServerName:         host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start email TLS client: %v", err)
	}

	// Authenticate
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate SMTP client: %v", err)
	}

	// Set the sender and recipient
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set email sender: %v", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set email recipient: %v", err)
		}
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get email writer: %v", err)
	}
	_, err = writer.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write email body: %v", err)
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close email writer: %v", err)
	}

	return nil
}

func formatEmailWithName(name string, email string) string {
	return fmt.Sprintf("%s <%s>", name, email)
}
