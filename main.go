package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Change send grid api
var SENDGRID_API_KEY = "SENDGRIDAPIHERE"
var wg sync.WaitGroup
var emailChan = make(chan Email)
var attachedFile string

type Email struct {
	Name string
	Address string
}

func main()  {
	readFile()
	// email_list format: <name>,<email>
	emails, err := os.Open("email_list.csv")
	if err != nil {
		log.Panicln("Couldn't open the csv file", err)
	}

	r := csv.NewReader(emails)
	poolSize := 10

	for i := 1; i <= poolSize; i++ {
		go sendEmail()
	}
	wg.Add(poolSize)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}

		emailChan<-Email{
			Name:    record[0],
			Address: record[1],
		}
	}

	for i := 1; i <= poolSize; i++ {
		emailChan<-Email{}
	}

	wg.Wait()
}

func sendEmail()  {
	request := sendgrid.GetRequest(SENDGRID_API_KEY, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	for {
		select {
		case email :=<-emailChan:
			if (Email{}) == email {
				wg.Done()
				log.Println("routine is closed")
				return
			}
			var Body = emailTemplate(email)
			request.Body = Body
			response, err := sendgrid.API(request)
			if err != nil {
				log.Fatal(err)
			} else {
				log.Printf("%s %d", email.Name, response.StatusCode)
			}
		}
	}
}

func emailTemplate(email Email) []byte {
	m := mail.NewV3Mail()

	address := "info@ftasymposium.sg"
	name := "FTA Symposium 2019 Information"
	e := mail.NewEmail(name, address)
	m.SetFrom(e)

	m.SetTemplateID("d-02f0b15fccbf489bb0a93e1b828284e8")

	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(email.Name, strings.TrimSpace(email.Address)),
	}
	p.AddTos(tos...)

	p.SetDynamicTemplateData("name", email.Name)

	a := mail.NewAttachment()
	a.SetContent(attachedFile)
	a.SetType("application/pdf")
	a.SetFilename("FTA Symposium 2019 Programme and Venue.pdf")
	a.SetDisposition("attachment")
	a.SetContentID("FTA Symposium 2019 Programme and Venue")
	m.AddAttachment(a)

	m.AddPersonalizations(p)
	return mail.GetRequestBody(m)
}

func readFile() {
	file, err := os.Open("FTA Symposium 2019 Programme and Venue.pdf")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()

	fInfo, _ := file.Stat()
	var size int64 = fInfo.Size()
	buf := make([]byte, size)

	fReader := bufio.NewReader(file)
	fReader.Read(buf)
	attachedFile = base64.StdEncoding.EncodeToString(buf)
}
