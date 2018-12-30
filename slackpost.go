package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	AlarmName        string  `json:"AlarmName"`
	AlarmDescription string  `json:"AlarmDescription"`
	NewStateValue    string  `json:"NewStateValue"`
	NewStateReason   string  `json:"NewStateReason"`
	OldStateValue    string  `json:"OldStateValue"`
	Region           string  `json:"Region"`
	Trigger          Trigger `json:"Trigger"`
}

type Trigger struct {
	Dimensions []Dimension `json:"Dimensions"`
}

type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Pretext    string  `json:"pretext"`
	Title      string  `json:"title"`
	TitleLink  string  `json:"title_link"`
	Text       string  `json:"text"`
	Color      string  `json:"color"`
	AuthorName string  `json:"author_name"`
	Fields     []Field `json:"fields"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

var AttachmentColor = map[string]string{
	"ALARM":             "danger",
	"INSUFFICIENT_DATA": "warning",
	"OK":                "good",
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
	attachment_fields := []Field{
		Field{
			Title: "Region",
			Value: message.Region,
			Short: true,
		},
		Field{
			Title: "Previous State",
			Value: message.OldStateValue,
			Short: true,
		},
	}

	for i := 0; i < len(message.Trigger.Dimensions); i++ {
		dimension := message.Trigger.Dimensions[i]
		attachment_fields = append(attachment_fields, Field{Title: dimension.Name, Value: dimension.Value, Short: true})
	}

	return SlackMessage{
		Attachments: []Attachment{
			Attachment{
				Pretext:   fmt.Sprintf("`%s`", message.AlarmDescription),
				Title:     fmt.Sprintf("%s: %s", message.NewStateValue, message.AlarmName),
				TitleLink: "https://console.aws.amazon.com/cloudwatch/home",
				Text:      message.NewStateReason,
				Color:     AttachmentColor[message.NewStateValue],
				Fields:    attachment_fields,
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
