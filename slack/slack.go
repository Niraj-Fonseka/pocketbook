package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pocketbook/store"
	"strings"

	"github.com/slack-go/slack"
	slacklib "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type SlackInterface interface {
}

type Slack struct {
	Store  store.StoreInterface
	API    *slack.Client
	Client *socketmode.Client
}

type SlackResponse struct {
	ResponseType   string `json:"response_type"`
	Text           string `json:"text,omitempty"`
	DeleteOriginal bool   `json:"delete_original"`
}

func NewSlack(driver string, api *slack.Client, client *socketmode.Client) *Slack {
	s := store.NewStore("dynamodb")
	return &Slack{
		Store: s,
	}
}

/*
	- middleware functions for hanlding slash commands

*/
func (s *Slack) middlewareSlashCommand(evt *socketmode.Event, client *socketmode.Client) {

}

func (s *Slack) middlewareInteractive(evt *socketmode.Event, client *socketmode.Client) {

}

func (s *Slack) MiddlewareInteractiveHandler(evt *socketmode.Event, client *socketmode.Client) {
	s.EventHandler(evt)
}

func (s *Slack) EventHandler(event *socketmode.Event) {
	eventCallBack := event.Data.(slack.InteractionCallback)

	switch eventCallBack.Type {
	case slacklib.InteractionTypeBlockActions:
		s.buttonClickHandler(event)
	case slacklib.InteractionTypeShortcut:
		//TODO: for future expansion but currently not implemented
	case slacklib.InteractionTypeViewSubmission:
		//TODO: for future expansion but currently not implemented
	default:
	}

}

func (s *Slack) slashCommandHandler(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	client.Debugf("Slash command received: %+v", cmd)

	fmt.Println("this is the text -----", cmd.Text)

	if strings.TrimSpace(cmd.Text) == "remove" {
		doc, err := s.Store.Get(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID))
		if err != nil {
			log.Println(err)
			return
		}

		mapDoc := doc.Data()["data"].([]interface{})
		client.Ack(*evt.Request, s.buildPayload(mapDoc, "delete"))
	}

	if len(strings.TrimSpace(cmd.Text)) > 0 {
		//add something
		client.Ack(*evt.Request)

		err := s.Store.Create(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID), strings.TrimSpace(cmd.Text))

		log.Println(err)
	} else {

		doc, err := s.Store.Get(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID))
		if err != nil {
			log.Println(err)
			return
		}

		mapDoc := doc.Data()["data"].([]interface{})
		client.Ack(*evt.Request, s.buildPayload(mapDoc, "send"))
	}
}

func (s *Slack) buildPayload(records []interface{}, event string) map[string]interface{} {

	var blocks []slack.Block

	for _, r := range records {
		blk := slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: "mrkdwn",
				Text: r.(string),
			},
			nil,
			slack.NewAccessory(
				slack.NewButtonBlockElement(
					"",
					r.(string),
					&slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: event,
					},
				),
			),
		)

		blocks = append(blocks, blk)
	}

	return map[string]interface{}{"blocks": blocks}
}

func (s *Slack) actionHandler(event *socketmode.Event, action string, payload string, responseURL string) {
	switch action {
	case "delete":
		s.deleteActionTrigger(event.Data.(slack.InteractionCallback), payload, responseURL)
	case "send":
		s.sendActionTrigger(event.Data.(slack.InteractionCallback), payload, responseURL)
	}

	s.Client.Ack(*event.Request, payload)

}

func (s *Slack) sendActionTrigger(event slack.InteractionCallback, payload string, responseURL string) {
	var slackResponse SlackResponse

	//https://api.slack.com/interactivity/slash-commands#responding_to_commands
	slackResponse.ResponseType = "in_channel"
	slackResponse.Text = fmt.Sprintf("post from @%s - %s", event.User.Name, payload)
	slackResponse.DeleteOriginal = true

	slackBytes, err := json.Marshal(&slackResponse)
	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Post(responseURL, "application/json", bytes.NewBuffer(slackBytes))

	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("Something went wrong status code : ", res.StatusCode)
	}
}

func (s *Slack) deleteActionTrigger(event slack.InteractionCallback, payload string, responseURL string) {

	err := s.Store.Delete(fmt.Sprintf("%s-%s", event.User.ID, event.Team.ID), dataToSend)
	if err != nil {
		log.Println(err)
	}

	var slackResponse SlackResponse

	slackResponse.ResponseType = "in_channel"
	slackResponse.DeleteOriginal = true

	slackBytes, err := json.Marshal(&slackResponse)
	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Post(responseURL, "application/json", bytes.NewBuffer(slackBytes))

	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("Something went wrong status code : ", res.StatusCode)
	}
	return
}

func (s *Slack) buttonClickHandler(event *socketmode.Event) {
	//https://api.slack.com/apis/connections/socket-implement#button

	b, ok := event.Data.(slack.InteractionCallback)

	if !ok {
		log.Println("unable to convert to a SlackCommand type")
		return
	}
	data := make(map[string]interface{})

	bts, err := json.Marshal(event.Data)

	if err != nil {
		log.Println("unable to convert to a json type")
		return
	}
	err = json.Unmarshal(bts, &data)
	if err != nil {
		log.Fatal(err)
	}

	responseURL := b.ResponseURL
	dataToSend := b.ActionCallback.BlockActions[0].Value
	action := b.ActionCallback.BlockActions[0].Text.Text //name of the button

	s.actionHandler(event, action, dataToSend, responseURL)

}
