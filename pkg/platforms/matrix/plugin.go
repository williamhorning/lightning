// Package matrix provides a [lightning.Plugin] implementation for Matrix.
// To use Matrix support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("matrix", matrix.New)
//
//	bot.UsePluginType("matrix", map[string]any{
//		// ...
//	})
package matrix

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// New creates a new [lightning.Plugin] that provides Matrix support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"access_token": "", // a string with your Matrix bot's token.
//						    // note: this should be set after initial login
//		"device_id": "", // a string with your Matrix bot's device ID.
//					     // note: this should be set after initial login
//		"homeserver": "",  // a string with your Matrix homeserver URL.
//						   // note: this MUST be set
//		"mxid": "",  // a string with your Matrix homeserver URL.
//					 // note: this should be set after initial login
//		"password": "", // a string with your Matrix bot password
//					    // note: this MUST be set
//		"random": "", // a random encryption key which MUST be set
//		"recovery_key": "", // a string with your Matrix bot recovery key
//					        // note: this MUST be set
//		"username": "", // a string with your Matrix bot username
//					    // note: this MUST be set
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid config"}
	}

	token, ok := cfg["access_token"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid access_token"}
	}

	device, ok := cfg["device_id"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid device_id"}
	}

	password, ok := cfg["password"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid password"}
	}

	username, ok := cfg["username"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid username"}
	}

	homeserver, ok := cfg["homeserver"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid homeserver"}
	}

	recoveryKey, ok := cfg["recovery_key"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid recovery_key"}
	}

	mxid, ok := cfg["mxid"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid mxid"}
	}

	random, ok := cfg["random"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid random"}
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(mxid), token)
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to create client: %w", err)
	}

	client.UserAgent = "lightning/" + lightning.VERSION

	if token == "" || device == "" || mxid == "" {
		resp, err := client.Login(context.Background(), &mautrix.ReqLogin{
			Type:             mautrix.AuthTypePassword,
			Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: username},
			Password:         password,
			StoreCredentials: true,
		})
		if err != nil {
			return nil, fmt.Errorf("matrix: failed to login: %w", err)
		}

		device = resp.DeviceID.String()
		token = resp.AccessToken
		mxid = resp.UserID.String()

		slog.Info("please set the following in your config:", "device_id", device, "access_token", token, "mxid", mxid)
	}

	helper, err := cryptohelper.NewCryptoHelper(
		client,
		[]byte(random),
		crypto.NewMemoryStore(func() error { return nil }),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup crypto helper: %w", err)
	}

	err = helper.Init(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to init crypto helper: %w", err)
	}

	client.Crypto = helper

	keyID, keyData, err := helper.Machine().SSSS.GetDefaultKeyData(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get default key: %w", err)
	}

	key, err := keyData.VerifyRecoveryKey(keyID, recoveryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify recovery key: %w", err)
	}

	err = helper.Machine().FetchCrossSigningKeysFromSSSS(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cross signing keys: %w", err)
	}

	err = helper.Machine().SignOwnDevice(context.Background(), helper.Machine().OwnIdentity())
	if err != nil {
		return nil, fmt.Errorf("failed to sign own device: %w", err)
	}

	err = helper.Machine().SignOwnMasterKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to sign own master key: %w", err)
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		client.Syncer = mautrix.NewDefaultSyncer()
	}

	msgChannel := make(chan *lightning.Message, 1000)
	editChannel := make(chan *lightning.EditedMessage, 1000)

	setupEvents(syncer, client, msgChannel, editChannel)

	return &matrixPlugin{
		client, syncer, cache.New[string, id.ContentURIString](cache.DefaultTTL),
		msgChannel, editChannel,
	}, nil
}

type matrixPlugin struct {
	client *mautrix.Client
	syncer *mautrix.DefaultSyncer

	mxcCache *cache.Expiring[string, id.ContentURIString]

	msgChannel  chan *lightning.Message
	editChannel chan *lightning.EditedMessage
}

func (*matrixPlugin) SetupChannel(_ string) (any, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *matrixPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	_ string,
) ([]string, error) {
	return p.SendMessage(message, opts)
}

func (p *matrixPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	ids := make([]string, 0, len(message.Attachments)+1)

	for _, msg := range p.getOutgoing(message, nil, opts) {
		resp, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return nil, handleError(err, "failed to send matrix message",
				map[string]any{"channel": message.ChannelID, "content": message.Content})
		}

		ids = append(ids, string(resp.EventID))
	}

	return ids, nil
}

func (p *matrixPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	for idx, msg := range p.getOutgoing(message, ids, opts) {
		msg.RelatesTo.SetReplace(id.EventID(ids[idx]))

		_, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return handleError(err, "failed to edit matrix message",
				map[string]any{"channel": message.ChannelID, "content": message.Content})
		}
	}

	return nil
}

func (p *matrixPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		if _, err := p.client.RedactEvent(
			context.Background(), id.RoomID(channel), id.EventID(msgID), mautrix.ReqRedact{Reason: "deleted in bridge"},
		); err != nil {
			return handleError(err, "Failed to redact Matrix message",
				map[string]any{"channel": channel, "message_id": msgID})
		}
	}

	return nil
}

func (*matrixPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *matrixPlugin) ListenMessages() <-chan *lightning.Message {
	return p.msgChannel
}

func (p *matrixPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	return p.editChannel
}

func (p *matrixPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	p.syncer.OnEventType(event.EventRedaction, func(_ context.Context, evt *event.Event) {
		timestamp := time.UnixMilli(evt.Timestamp)

		channel <- &lightning.BaseMessage{
			Time:      &timestamp,
			EventID:   evt.Content.AsRedaction().Redacts.String(),
			ChannelID: evt.RoomID.String(),
		}
	})

	return channel
}

func (*matrixPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
