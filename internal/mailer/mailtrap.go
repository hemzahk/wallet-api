package mailer

import (
	"bytes"
	"errors"
	"text/template"

	gomail "gopkg.in/mail.v2"
)

type mailtrapClient struct {
	fromEmail string
	username    string
	password string
}

func NewMailTrapClient(username, password, fromEmail string) (mailtrapClient, error) {
	if username == "" || password == "" {
		return mailtrapClient{}, errors.New("credentials are required")
	}

	return mailtrapClient{
		fromEmail: fromEmail,
		username: username,
		password: password,
	}, nil
}

func (m mailtrapClient) Send(templateFile, firstName, email string, data any, isSandBox bool) (int, error) {
	// parse template
	tmpl, err := template.ParseFS(FS, "templates/"+templateFile)
	if err != nil {
		return -1, err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return -1, err
	}

	body := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(body, "body", data)
	if err != nil {
		return -1, err
	}

	message := gomail.NewMessage()
	message.SetHeader("From", m.fromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", subject.String())

	message.AddAlternative("text/html", body.String())

	dialer := gomail.NewDialer("sandbox.smtp.mailtrap.io", 2525, m.username, m.password)

	if err := dialer.DialAndSend(message); err != nil {
		return -1, err
	}

	return 200, nil
} 