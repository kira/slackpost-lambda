package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

type Request struct {
	Records []struct {
		SNS struct {
			Type       string `json:"Type"`
			Timestamp  string `json:"Timestamp"`
			SNSMessage string `json:"Message"`
		} `json:"Sns"`
	} `json:"Records"`
}

type SNSMessage struct {
	AlarmName        string `json:"AlarmName"`
	AlarmDescription string `json:"AlarmDescription"`
	NewStateValue    string `json:"NewStateValue"`
	NewStateReason   string `json:"NewStateReason"`
	Region           string `json:"Region"`
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Title       string       `json:"title"`
	TitleUrl    string       `json:"title_url"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}

func handler(request Request) error {
	var snsMessage SNSMessage
	err := json.Unmarshal([]byte(request.Records[0].SNS.SNSMessage), &snsMessage)
	if err != nil {
		return err
	}

	log.Printf("New alarm: %s - Reason: %s", snsMessage.AlarmName, snsMessage.NewStateReason)
	slackMessage := buildSlackMessage(snsMessage)
	postToSlack(slackMessage)
	log.Println("Notification has been sent")
	return nil
}

func buildSlackMessage(message SNSMessage) SlackMessage {
	return SlackMessage{
		Text: fmt.Sprintf("`%s`", message.AlarmDescription),
		Title: fmt.Sprintf(
			"<https://console.aws.amazon.com/cloudwatch/home#s=%s|%s>",
			url.PathEscape(message.AlarmName), message.AlarmName,
		),
		Attachments: []Attachment{
			Attachment{
				Text:  message.NewStateReason,
				Color: "danger",
				Title: "Reason",
			},
			Attachment{
				Text:  message.Region,
				Color: "default",
				Title: "Region",
			},
		},
	}
}

func postToSlack(message SlackMessage) error {
	client := &http.Client{}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", os.Getenv("SLACK_WEBHOOK"), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println(resp.StatusCode)
		return err
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
