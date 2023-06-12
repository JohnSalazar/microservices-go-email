package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/JohnSalazar/microservices-go-common/config"
	trace "github.com/JohnSalazar/microservices-go-common/trace/otel"
	mail "github.com/xhit/go-simple-mail/v2"
)

var path = "src/www/templates/"

type emailService struct {
	config *config.Config
}

func NewEmailService(
	config *config.Config,
) *emailService {
	return &emailService{
		config: config,
	}
}

func (c *emailService) SendPasswordCode(email string, code string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Second*10)
	defer close()

	ctx, span := trace.NewSpan(ctx, "emailService.SendPasswordCode")
	defer span.End()

	smtpServer, err := c.createSMTPServer(ctx)
	if err != nil {
		log.Print(fmt.Errorf("error create SMTP Server: %s", err))
		return err
	}

	log.Print("create SMTP Server")

	smtpClient, err := smtpServer.Connect()
	if err != nil {
		log.Print(fmt.Errorf("error connect to smtp: %s", err.Error()))
		return fmt.Errorf("error connect to smtp: %s", err.Error())
	}
	defer smtpClient.Close()

	log.Print("connect to smtp")

	err = smtpClient.Noop()
	if err != nil {
		return err
	}

	filenameHTML := path + "code-template.html"
	data := c.dataCodeGenerate(code)
	htmlBody := new(bytes.Buffer)

	html, err := template.ParseFiles(filenameHTML)
	if err != nil {
		log.Print(fmt.Errorf("error parse html file: %s", err.Error()))
		return fmt.Errorf("error parse html file: %s", err.Error())
	}

	log.Print("parse html file")

	err = html.Execute(htmlBody, data)
	if err != nil {
		log.Print(fmt.Errorf("error generate template: %s", err.Error()))
		return fmt.Errorf("error generate template: %s", err.Error())
	}

	log.Print("generate template")

	from := fmt.Sprintf("%s <%s>", c.config.Company.Name, c.config.Company.Email)

	emailMSG := mail.NewMSG()
	emailMSG.SetFrom(from)
	emailMSG.AddTo(email)
	emailMSG.SetSubject("Request to change password")
	emailMSG.SetBody(mail.TextHTML, htmlBody.String())

	err = emailMSG.Send(smtpClient)
	if err != nil {
		log.Print(fmt.Errorf("error to send email: %s", err.Error()))
		log.Print(emailMSG)
		return fmt.Errorf("error to send email: %s", err.Error())
	}

	log.Print("send email")

	return nil
}

func (c *emailService) SendSupportMessage(message string) error {
	ctx, close := context.WithTimeout(context.Background(), time.Second*10)
	defer close()

	ctx, span := trace.NewSpan(ctx, "emailService.SendSupportMessage")
	defer span.End()

	smtpServer, err := c.createSMTPServer(ctx)
	if err != nil {
		return err
	}

	smtpClient, err := smtpServer.Connect()
	if err != nil {
		return fmt.Errorf("error connect to smtp: %s", err.Error())
	}

	filenameHTML := path + "support-template.html"
	data := c.dataMessageGenerate(message)
	htmlBody := new(bytes.Buffer)

	html, err := template.ParseFiles(filenameHTML)
	if err != nil {
		return fmt.Errorf("error parse html file: %s", err.Error())
	}

	err = html.Execute(htmlBody, data)
	if err != nil {
		return fmt.Errorf("error generate template: %s", err.Error())
	}

	from := fmt.Sprintf("%s <%s>", c.config.Company.Name, c.config.SMTPServer.SupportEmail)

	emailMSG := mail.NewMSG()
	emailMSG.SetFrom(from)
	emailMSG.AddTo(c.config.Company.Email)
	emailMSG.SetSubject("Support")
	emailMSG.SetBody(mail.TextHTML, htmlBody.String())

	err = emailMSG.Send(smtpClient)
	if err != nil {
		return fmt.Errorf("error to send email: %s", err.Error())
	}

	return nil
}

func (c *emailService) createSMTPServer(ctx context.Context) (*mail.SMTPServer, error) {
	_, span := trace.NewSpan(ctx, "emailService.createSMTPServer")
	defer span.End()

	smtpServer := mail.NewSMTPClient()
	smtpServer.Authentication = mail.AuthLogin
	smtpServer.Host = c.config.SMTPServer.Host
	smtpServer.Port = c.config.SMTPServer.Port
	smtpServer.Username = c.config.SMTPServer.Username
	smtpServer.Password = c.config.SMTPServer.Password
	smtpServer.ConnectTimeout = 10 * time.Second
	smtpServer.SendTimeout = 10 * time.Second
	smtpServer.KeepAlive = false

	if c.config.SMTPServer.TLS {
		smtpServer.Encryption = mail.EncryptionTLS
		smtpServer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		smtpServer.Encryption = mail.EncryptionNone
	}

	return smtpServer, nil
}

func (c *emailService) dataCodeGenerate(code string) interface{} {
	type dataTemplate struct {
		Code              string
		ImgLogo           string
		ImgPadlock        string
		ImgFacebook       string
		ImgTwitter        string
		ImgInstagram      string
		ImgLinkedin       string
		LinkFacebook      string
		LinkTwitter       string
		LinkInstagram     string
		LinkLinkedin      string
		Name              string
		Address           string
		AddressNumber     string
		AddressComplement string
		Phone             string
		Email             string
	}

	data := dataTemplate{
		Code:              code,
		ImgLogo:           "https://iili.io/l62C2p.png",
		ImgPadlock:        "https://iili.io/l62ITX.png",
		ImgFacebook:       "https://iili.io/l62ovI.png",
		ImgTwitter:        "https://iili.io/l62Tjn.png",
		ImgInstagram:      "https://iili.io/l62nYN.png",
		ImgLinkedin:       "https://iili.io/l62xpt.png",
		LinkFacebook:      "https://www.facebook.com/joao.salazar.5/",
		LinkTwitter:       "https://twitter.com/joaosalazar",
		LinkInstagram:     "https://www.instagram.com/johnsalazar",
		LinkLinkedin:      "https://www.linkedin.com/in/joao-salazar-67237365/",
		Name:              c.config.Company.Name,
		Address:           c.config.Company.Address,
		AddressNumber:     c.config.Company.AddressNumber,
		AddressComplement: c.config.Company.AddressComplement,
		Phone:             c.config.Company.Phone,
		Email:             c.config.Company.Email,
	}

	return data
}

func (c *emailService) dataMessageGenerate(message string) interface{} {
	type dataTemplate struct {
		Message string
	}

	data := dataTemplate{
		Message: message,
	}

	return data
}
