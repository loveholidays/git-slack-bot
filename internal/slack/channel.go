package slack

func (sc *Connector) Channel(channelID string) Interactor {
	return &Connector{client: sc.client, channelID: channelID, notificationMutex: sc.notificationMutex}
}

func (sc *Connector) ConversationID() string {
	return sc.channelID
}
