package pocketbook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Niraj-Fonseka/threedb"

	"github.com/slack-go/slack"
	slacklib "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"go.uber.org/zap"
)

type SlackInterface interface {
}

type Slack struct {
	store  threedb.ThreeDBInterface
	api    *slack.Client
	client *socketmode.Client
	logger *zap.SugaredLogger
}

type SlackResponse struct {
	ResponseType   string `json:"response_type"`
	Text           string `json:"text,omitempty"`
	DeleteOriginal bool   `json:"delete_original"`
}

func NewPockebookClient(driver string, api *slack.Client, client *socketmode.Client, logger *zap.SugaredLogger) *Slack {
	s := threedb.NewThreeDB(driver)
	return &Slack{
		store:  s,
		api:    api,
		client: client,
		logger: logger,
	}
}

/*
	- middleware functions for hanlding slash commands
*/

func (s *Slack) MiddlewareConnecting(evt *socketmode.Event, client *socketmode.Client) {
	s.logger.Info("Connecting to Slack with Socket Mode...")
}

func (s *Slack) MiddlewareConnectionError(evt *socketmode.Event, client *socketmode.Client) {
	s.logger.Info("Connection failed. Retrying later...")
}

func (s *Slack) MiddlewareConnected(evt *socketmode.Event, client *socketmode.Client) {
	s.logger.Info("Connected to Slack with Socket Mode.")
}

func (s *Slack) MiddlewareSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	s.Error(s.slashCommandHandler(evt, client))
}

func (s *Slack) MiddlewareInteractive(evt *socketmode.Event, client *socketmode.Client) {
	s.Error(s.EventHandler(evt))
}

func (s *Slack) EventHandler(event *socketmode.Event) error {
	eventCallBack := event.Data.(slack.InteractionCallback)

	switch eventCallBack.Type {
	case slacklib.InteractionTypeBlockActions:
		return s.buttonClickHandler(event)
	case slacklib.InteractionTypeShortcut:
		//TODO: for future expansion but currently not implemented
	case slacklib.InteractionTypeViewSubmission:
		//TODO: for future expansion but currently not implemented
	default:
	}

	return nil
}

func (s *Slack) getSlashCommandHandler(event *socketmode.Event) error {

	eventData, ok := event.Data.(slack.SlashCommand)
	if !ok {
		return fmt.Errorf("ignored event %+v", event)
	}

	doc, err := s.store.Get(fmt.Sprintf("%s-%s", eventData.UserID, eventData.TeamID))
	if err != nil {
		return err
	}

	mapDoc := doc.Data()["data"].([]interface{})
	s.client.Ack(*event.Request, s.buildPayload(mapDoc, "send"))

	return nil
}

func (s *Slack) createSlashCommandHandler(event *socketmode.Event) error {

	eventData, ok := event.Data.(slack.SlashCommand)
	if !ok {
		return fmt.Errorf("ignored event %+v", event)
	}

	if len(strings.TrimSpace(eventData.Text)) > 0 {
		//add something
		s.client.Ack(*event.Request)

		err := s.store.Create(fmt.Sprintf("%s-%s", eventData.UserID, eventData.TeamID), strings.TrimSpace(eventData.Text))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Slack) deleteSlashCommandHandler(event *socketmode.Event) error {
	eventData, ok := event.Data.(slack.SlashCommand)
	if !ok {
		return fmt.Errorf("ignored event %+v", event)
	}

	doc, err := s.store.Get(fmt.Sprintf("%s-%s", eventData.UserID, eventData.TeamID))
	if err != nil {
		return err
	}

	mapDoc := doc.Data()["data"].([]interface{})
	s.client.Ack(*event.Request, s.buildPayload(mapDoc, "delete"))
	return nil
}

func (s *Slack) slashCommandHandler(event *socketmode.Event, client *socketmode.Client) error {
	eventData, ok := event.Data.(slack.SlashCommand)
	if !ok {
		return fmt.Errorf("ignored event %+v", event)
	}

	if strings.TrimSpace(eventData.Text) == "delete" {
		s.Error(s.deleteSlashCommandHandler(event))
	} else if len(strings.TrimSpace(eventData.Text)) > 0 {
		s.Error(s.createSlashCommandHandler(event))
	} else {
		s.Error(s.getSlashCommandHandler(event))
	}
	return nil
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

func (s *Slack) eventHandler(event *socketmode.Event, action string, eventType string, payload string) {

	switch eventType {
	case "slash":
		s.Error(s.slashCommandHandler(event, s.client))
	case "button":
		s.Error(s.buttonClickHandler(event))
	}

	s.client.Ack(*event.Request, payload)

}

func (s *Slack) sendButtonClickHandler(event slack.InteractionCallback, payload string, responseURL string) error {
	var slackResponse SlackResponse

	//https://api.slack.com/interactivity/slash-commands#responding_to_commands
	slackResponse.ResponseType = "in_channel"
	slackResponse.Text = fmt.Sprintf("post from @%s - %s", event.User.Name, payload)
	slackResponse.DeleteOriginal = true

	slackBytes, err := json.Marshal(&slackResponse)
	if err != nil {
		return err
	}

	res, err := http.Post(responseURL, "application/json", bytes.NewBuffer(slackBytes))

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Something went wrong with sending the webhook status_code : ", res.StatusCode)
	}
	return nil
}

func (s *Slack) deleteButtonClickHandler(event slack.InteractionCallback, payload string, responseURL string) error {

	err := s.store.Delete(fmt.Sprintf("%s-%s", event.User.ID, event.Team.ID), payload)
	if err != nil {
		return err
	}

	var slackResponse SlackResponse

	slackResponse.ResponseType = "in_channel"
	slackResponse.DeleteOriginal = true

	slackBytes, err := json.Marshal(&slackResponse)
	if err != nil {
		return err
	}

	res, err := http.Post(responseURL, "application/json", bytes.NewBuffer(slackBytes))

	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Something went wrong with sending the webhook status_code : ", res.StatusCode)
	}
	return nil
}

func (s *Slack) buttonClickHandler(event *socketmode.Event) error {
	//https://api.slack.com/apis/connections/socket-implement#button

	eventData, ok := event.Data.(slack.InteractionCallback)

	if !ok {
		return fmt.Errorf("ignored event %+v", event)
	}

	data := make(map[string]interface{})

	bts, err := json.Marshal(event.Data)

	if err != nil {
		return err
	}
	err = json.Unmarshal(bts, &data)
	if err != nil {
		return err
	}

	responseURL := eventData.ResponseURL
	payload := eventData.ActionCallback.BlockActions[0].Value
	action := eventData.ActionCallback.BlockActions[0].Text.Text

	switch action {
	case "delete":
		s.Error(s.deleteButtonClickHandler(event.Data.(slack.InteractionCallback), payload, responseURL))
	case "send":
		s.Error(s.sendButtonClickHandler(event.Data.(slack.InteractionCallback), payload, responseURL))
	}

	return nil

}

func (s *Slack) Error(err error) {
	if err != nil {
		s.logger.Error(err)
	}
}
