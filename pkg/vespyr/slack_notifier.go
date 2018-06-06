package vespyr

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

var slackNotifier *SlackNotifier

// SlackNotifier posts messages to a single Slack client.
type SlackNotifier struct {
	tradesChannel string
	dataChannel   string
	client        *slack.Client
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(trades, data string, client *slack.Client) *SlackNotifier {
	return &SlackNotifier{
		tradesChannel: trades,
		dataChannel:   data,
		client:        client,
	}
}

// PostStrategyDataToSlack posts information about a strategy's tick
// to Slack.
func PostStrategyDataToSlack(strategy StrategyInterface, ts *TradingStrategyModel, meta map[string]interface{}) {
	message := fmt.Sprintf(`Strategy: %s
ID: %d
State: %s
Product: %s`, strategy, ts.ID, ts.State, ts.Product)

	for key, value := range meta {
		message = fmt.Sprintf("%s\n%s: %v", message, key, value)
	}

	PostDataSlackMessage("", slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{
			{Title: "Tick", Text: message, Color: "#b942f4"},
		},
	})
}

// PostTradesSlackMessage posts a Slack message to the configured channel.
func PostTradesSlackMessage(text string, params ...slack.PostMessageParameters) {
	if slackNotifier == nil {
		return
	}
	if len(params) == 0 {
		params = []slack.PostMessageParameters{slack.PostMessageParameters{AsUser: true}}
	}
	_, _, err := slackNotifier.client.PostMessage(slackNotifier.tradesChannel, text, params[0])
	if err != nil {
		logrus.WithError(err).Errorf("error posting messages to Slack")
	}
}

// PostDataSlackMessage posts a Slack message to the configured channel.
func PostDataSlackMessage(text string, params ...slack.PostMessageParameters) {
	if slackNotifier == nil {
		return
	}
	if len(params) == 0 {
		params = []slack.PostMessageParameters{slack.PostMessageParameters{AsUser: true}}
	}
	_, _, err := slackNotifier.client.PostMessage(slackNotifier.dataChannel, text, params[0])
	if err != nil {
		logrus.WithError(err).Errorf("error posting messages to Slack")
	}
}
