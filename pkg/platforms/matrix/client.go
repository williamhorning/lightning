package matrix

import (
	"context"
	"fmt"
	"log"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func getBot(config map[string]string) (lightning.Plugin, error) {
	client, err := mautrix.NewClient(config["homeserver"], id.UserID(config["mxid"]), config["access_token"])
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		syncer = mautrix.NewDefaultSyncer()
		client.Syncer = syncer
	}

	msgChannel := make(chan *lightning.Message, 1000)
	editChannel := make(chan *lightning.EditedMessage, 1000)
	deleteChannel := make(chan *lightning.BaseMessage, 1000)

	syncer.OnSync(client.DontProcessOldEvents)

	go func() {
		for {
			if err := client.Sync(); err != nil {
				log.Printf("matrix: sync stopped, will retry: %v\n", err)
			}
		}
	}()

	go startProxy(client, config["proxy_url"], config["proxy_port"])

	plugin := &matrixPlugin{
		proxy: config["proxy_url"], client: client, msgChannel: msgChannel,
		editChannel: editChannel, deleteChannel: deleteChannel,
	}

	plugin.listenForEvents(
		syncer.OnEventType, client.JoinRoomByID, client, string(client.UserID),
		msgChannel, editChannel, deleteChannel,
	)

	log.Println("matrix: bot ready at https://matrix.to/#/" + config["mxid"])

	return plugin, nil
}

func getAppsvc(config map[string]string) (lightning.Plugin, error) {
	appsvc, err := appservice.CreateFull(appservice.CreateOpts{
		HomeserverDomain: config["homeserver"],
		HomeserverURL:    "https://" + config["homeserver"],
		HostConfig:       appservice.HostConfig{Port: 5000},
		Registration: &appservice.Registration{
			ID:              "lightning",
			URL:             config["as_url"],
			AppToken:        config["as_token"],
			ServerToken:     config["hs_token"],
			SenderLocalpart: config["as_local"],
			Namespaces: appservice.Namespaces{UserIDs: []appservice.Namespace{{
				Exclusive: true,
				Regex:     "@_lightning_.*",
			}}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appservice: %w", err)
	}

	handler := appservice.NewEventProcessor(appsvc)

	go appsvc.Start()
	go handler.Start(context.Background())
	go startProxy(appsvc.BotClient(), config["proxy_url"], config["proxy_port"])

	msgChannel := make(chan *lightning.Message, 1000)
	editChannel := make(chan *lightning.EditedMessage, 1000)
	deleteChannel := make(chan *lightning.BaseMessage, 1000)

	plugin := &matrixPlugin{
		proxy: config["proxy_url"], appsvc: appsvc, client: appsvc.BotClient(),
		msgChannel: msgChannel, editChannel: editChannel, deleteChannel: deleteChannel,
	}

	plugin.listenForEvents(
		func(eventType event.Type, callback mautrix.EventHandler) {
			handler.On(eventType, callback)
		}, appsvc.BotClient().JoinRoomByID, appsvc.BotClient(), `^@_lightning_.*:`+config["homeserver"],
		msgChannel, editChannel, deleteChannel,
	)

	log.Println("matrix: appservice ready at https://matrix.to/#/" + appsvc.BotMXID())

	return plugin, nil
}

func (p *matrixPlugin) getClient(msg *lightning.Message) (*mautrix.Client, bool) {
	if msg == nil || msg.Author == nil || p.appsvc == nil {
		return p.client, true
	}

	intent := p.appsvc.NewIntentAPI("_lightning_" + msg.Author.ID)

	if err := intent.EnsureJoined(context.Background(), id.RoomID(msg.ChannelID)); err != nil {
		return p.client, true
	}

	if err := intent.SetProfileField(context.Background(), "displayname", msg.Author.Username); err != nil {
		return intent.Client, false
	}

	if err := intent.SetAvatarURL(context.Background(),
		p.uploadFile(intent.Client, msg.Author.ProfilePicture).ParseOrIgnore()); err != nil {
		return intent.Client, false
	}

	return intent.Client, false
}
