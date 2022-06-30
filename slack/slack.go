package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pocketbook/store"

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

	s := store.NewStore("firestore")
	return &Slack{
		Store: s,
	}
}

func (s *Slack) MiddlewareInteractiveHandler(evt *socketmode.Event, client *socketmode.Client) {
	s.EventHandler(evt)
}

func (s *Slack) EventHandler(event *socketmode.Event) {
	eventCallBack := event.Data.(slack.InteractionCallback)

	switch eventCallBack.Type {
	case slacklib.InteractionTypeBlockActions:
		s.ButtonClickHandler(event)
	case slacklib.InteractionTypeShortcut:
		//TODO: for future expansion but currently implemented
	case slacklib.InteractionTypeViewSubmission:
		//TODO: for future expansion but currently implemented
	default:
	}

}

func (s *Slack) actionHandler(action string, event *socketmode.Event) {
	switch action {
	case "delete":
		s.deleteActionTrigger()
	case "send":

	}
}

func (s *Slack) sendActionTrigger(event *socketmode.Event) {

}

func (s *Slack) deleteActionTrigger(event *socketmode.Event) {
	s.Client.Ack(*event.Request, payload)

	err := s.Store.Delete(fmt.Sprintf("%s-%s", b.User.ID, b.Team.ID), dataToSend)
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

func (s *Slack) ButtonClickHandler(event *socketmode.Event) {
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

	fmt.Println("call backs ")

	responseURL := b.ResponseURL
	// dataToSend := data["actions"].([]interface{})[0].(map[string]interface{})["value"].(string)
	dataToSend := b.ActionCallback.BlockActions[0].Value
	action := b.ActionCallback.BlockActions[0].Text.Text //name of the button

	// if action == "delete" {
	// 	//trigger delete
	// 	s.Client.Ack(*event.Request, payload)

	// 	err := s.Store.Delete(fmt.Sprintf("%s-%s", b.User.ID, b.Team.ID), dataToSend)
	// 	if err != nil {
	// 		log.Println(err)
	// 	}

	// 	var slackResponse SlackResponse

	// 	slackResponse.ResponseType = "in_channel"
	// 	slackResponse.DeleteOriginal = true

	// 	slackBytes, err := json.Marshal(&slackResponse)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	res, err := http.Post(responseURL, "application/json", bytes.NewBuffer(slackBytes))

	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	if res.StatusCode != 200 {
	// 		log.Fatal("Something went wrong status code : ", res.StatusCode)
	// 	}
	// 	return
	// }

	var slackResponse SlackResponse

	fmt.Println(b.User)
	//https://api.slack.com/interactivity/slash-commands#responding_to_commands
	slackResponse.ResponseType = "in_channel"
	slackResponse.Text = fmt.Sprintf("post from @%s - %s", b.User.Name, dataToSend)
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
