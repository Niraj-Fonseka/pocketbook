package main

import (
	"log"
	"os"
	"strings"

	"github.com/Niraj-Fonseka/pocketbook/pocketbook"

	"go.uber.org/zap"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	sugar := logger.Sugar()
	defer logger.Sync()

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		sugar.Fatal("SLACK_APP_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		sugar.Fatal("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		sugar.Fatal("SLACK_BOT_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		sugar.Fatal("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
	)

	socketmodeHandler := socketmode.NewSocketmodeHandler(client)
	pocketbookConnector := pocketbook.NewPockebookClient("firestore", api, client, sugar)
	socketmodeHandler.Handle(socketmode.EventTypeConnecting, pocketbookConnector.MiddlewareConnecting)
	socketmodeHandler.Handle(socketmode.EventTypeConnectionError, pocketbookConnector.MiddlewareConnectionError)
	socketmodeHandler.Handle(socketmode.EventTypeConnected, pocketbookConnector.MiddlewareConnected)
	socketmodeHandler.Handle(socketmode.EventTypeInteractive, pocketbookConnector.MiddlewareInteractive)

	socketmodeHandler.HandleSlashCommand("/pocketbook", pocketbookConnector.MiddlewareSlashCommand)

	socketmodeHandler.RunEventLoop()
}
