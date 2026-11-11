package slack

import "github.com/slack-go/slack"

func (sc *Connector) DirectMessage(userID string) (Interactor, error) {
	channel, _, _, err := sc.client.OpenConversation(&slack.OpenConversationParameters{Users: []string{userID}})
	if err != nil {
		return nil, err
	}
	return &Connector{client: sc.client, channelID: channel.ID, notificationMutex: sc.notificationMutex}, nil
}
