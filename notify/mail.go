package notify

import (
	"fmt"
	"ruscraper/models"
	"net/smtp"
)

func SendNewThemesByMail(recepient string, themes []models.Theme) {
	auth := smtp.PlainAuth(
        "",
        "",
        "",
        "mail.google.com",
    )
    // Connect to the server, authenticate, set the sender and recipient,
    // and send the email all in one step.

	body := ""

	for i, t := range(themes) {
		body += fmt.Sprintf("\r\n%d) %s", i, t.Name)
	}

    err := smtp.SendMail(
        "mail.google.com:25",
        auth,
        "thinkandwin5000@google.com",
        []string{recepient},
        []byte(body),
    )

    fmt.Println("mail sended")
    if err != nil {
        fmt.Println("error on send mail", err)
    }
}
