package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"pocketbook/store"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"google.golang.org/api/option"

	"github.com/slack-go/slack"
)

type SlackResponse struct {
	ResponseType   string `json:"response_type"`
	Text           string `json:"text,omitempty"`
	DeleteOriginal bool   `json:"delete_original"`
}

var FS *store.FirestoreService

func main() {

	ctx := context.Background()
	sa := option.WithCredentialsFile("./pocketbook-firestore.json")

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	FS = store.NewFirestoreService(app, ctx)

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		panic("SLACK_APP_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		panic("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		panic("SLACK_BOT_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		panic("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		// slack.OptionDebug(true),
		// slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		// socketmode.OptionDebug(true),
		// socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	socketmodeHandler := socketmode.NewSocketmodeHandler(client)

	socketmodeHandler.Handle(socketmode.EventTypeConnecting, middlewareConnecting)
	socketmodeHandler.Handle(socketmode.EventTypeConnectionError, middlewareConnectionError)
	socketmodeHandler.Handle(socketmode.EventTypeConnected, middlewareConnected)
	socketmodeHandler.Handle(socketmode.EventTypeInteractive, middlewareInteractive)

	socketmodeHandler.HandleSlashCommand("/pocketbook", middlewareSlashCommand)

	// socketmodeHandler.HandleDefault(middlewareDefault)

	socketmodeHandler.RunEventLoop()
}

func middlewareConnecting(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connecting to Slack with Socket Mode...")
}

func middlewareConnectionError(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connection failed. Retrying later...")
}

func middlewareConnected(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connected to Slack with Socket Mode.")
}

func middlewareEventsAPI(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("middlewareEventsAPI")
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	fmt.Printf("Event received: %+v\n", eventsAPIEvent)

	client.Ack(*evt.Request)

	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent:
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			fmt.Printf("We have been mentionned in %v", ev.Channel)
			_, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			if err != nil {
				fmt.Printf("failed posting message: %v", err)
			}
		case *slackevents.MemberJoinedChannelEvent:
			fmt.Printf("user %q joined to channel %q", ev.User, ev.Channel)
		}
	default:
		client.Debugf("unsupported Events API event received")
	}
}

func middlewareAppMentionEvent(evt *socketmode.Event, client *socketmode.Client) {

	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	client.Ack(*evt.Request)

	ev, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.AppMentionEvent)
	if !ok {
		fmt.Printf("Ignored %+v\n", ev)
		return
	}

	fmt.Printf("We have been mentionned in %v\n", ev.Channel)
	_, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
	if err != nil {
		fmt.Printf("failed posting message: %v", err)
	}
}

func middlewareInteractive(evt *socketmode.Event, client *socketmode.Client) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	var payload interface{}

	switch callback.Type {
	case slack.InteractionTypeBlockActions:
		// See https://api.slack.com/apis/connections/socket-implement#button
		client.Debugf("button clicked!")
		fmt.Println("------------------------ button clicked !!")

		bts, err := json.Marshal(evt.Data)

		if err != nil {
			log.Println("unable to convert to a json type")
			return
		}

		b, ok := evt.Data.(slack.InteractionCallback)

		if !ok {
			log.Println("unable to convert to a SlackCommand type")
			return
		}

		fmt.Println("event data --------------------------------")
		data := make(map[string]interface{})

		err = json.Unmarshal(bts, &data)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("call backs ")

		responseURL := b.ResponseURL
		// dataToSend := data["actions"].([]interface{})[0].(map[string]interface{})["value"].(string)
		dataToSend := b.ActionCallback.BlockActions[0].Value
		action := b.ActionCallback.BlockActions[0].Text.Text //name of the button

		if action == "delete" {
			//trigger delete
			client.Ack(*evt.Request, payload)

			err := FS.DeleteRecord(fmt.Sprintf("%s-%s", b.User.ID, b.Team.ID), dataToSend)
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
	case slack.InteractionTypeShortcut:
	case slack.InteractionTypeViewSubmission:
		// See https://api.slack.com/apis/connections/socket-implement#modal
	case slack.InteractionTypeDialogSubmission:
	default:

	}

	client.Ack(*evt.Request, payload)
}

func middlewareInteractionTypeBlockActions(evt *socketmode.Event, client *socketmode.Client) {
	client.Debugf("button clicked!")
}

func middlewareSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	client.Debugf("Slash command received: %+v", cmd)

	fmt.Println("this is the text -----", cmd.Text)

	if strings.TrimSpace(cmd.Text) == "remove" {
		doc, err := FS.GetUserRecord(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID))
		if err != nil {
			log.Println(err)
			return
		}

		mapDoc := doc.Data()["data"].([]interface{})
		client.Ack(*evt.Request, buildPayload(mapDoc, "delete"))
	}

	if len(strings.TrimSpace(cmd.Text)) > 0 {
		//add something
		client.Ack(*evt.Request)

		err := FS.AddUserRecord(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID), strings.TrimSpace(cmd.Text))

		log.Println(err)
	} else {

		doc, err := FS.GetUserRecord(fmt.Sprintf("%s-%s", cmd.UserID, cmd.TeamID))
		if err != nil {
			log.Println(err)
			return
		}

		mapDoc := doc.Data()["data"].([]interface{})
		client.Ack(*evt.Request, buildPayload(mapDoc, "send"))
	}
}

func buildPayload(records []interface{}, event string) map[string]interface{} {

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
