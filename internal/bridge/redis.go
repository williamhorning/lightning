package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/williamhorning/lightning/pkg/lightning"
)

var ErrUnsupportedVersion = errors.New("unsupported database version, please upgrade to the latest version of Lightning")

const (
	keyVersion   = "lightning-db-version"
	keyBridge    = "lightning-bridge-"
	keyBChannel  = "lightning-bchannel-"
	keyMessage   = "lightning-message-"
	validVersion = "0.8.0"
)

type redisDatabase struct {
	rdb *redis.Client
	ctx context.Context
	svn bool
}

func newRedisDatabase(addr string) (Database, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	version, err := client.Get(ctx, keyVersion).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	} else if err == redis.Nil {
		keys, err := client.DBSize(ctx).Result()
		if err != nil {
			return nil, err
		}

		if keys > 0 {
			lightning.Log.Warn().Msg("Migrating from 0.7.x to 0.8.0, this may take a while")

			self := redisDatabase{client, ctx, true}

			bridges, err := self.GetAllBridges()

			if err != nil {
				return nil, err
			}

			jsonData, err := json.Marshal(bridges)

			if err != nil {
				return nil, err
			}

			err = os.WriteFile("lightning-redis-migration.json", jsonData, 0777)

			if err != nil {
				return nil, err
			}

			lightning.Log.Warn().Msg("Do you want to write the migrated data to the database? See lightning-redis-migration.json for the data to be written. [y/N]")

			b := make([]byte, 1)

			_, err = os.Stdin.Read(b)

			if err != nil {
				return nil, err
			}

			if !(os.Getenv("LIGHTNING_MIGRATE_CONFIG") != "" || b[0] == 'y') {
				lightning.Log.Warn().Msg("Migration aborted, please run the command again with LIGHTNING_MIGRATE_CONFIG=1 to write the data to the database")
				return nil, ErrUnsupportedVersion
			}

			lightning.Log.Info().Msg("Writing migrated data to the database")

			err = self.SetAllBridges(bridges)

			if err != nil {
				return nil, err
			}

			lightning.Log.Info().Msg("Migration completed successfully")

			self.svn = false

			return self, nil
		}

		if _, err := client.Set(ctx, keyVersion, validVersion, 0).Result(); err != nil {
			return nil, err
		}
		version = validVersion
	} else if version != validVersion {
		return nil, ErrUnsupportedVersion
	}

	return redisDatabase{client, ctx, false}, nil
}

func (r redisDatabase) createBridge(bridge Bridge) error {
	bridgeJSON, err := json.Marshal(bridge)
	if err != nil {
		return err
	}

	if val, err := r.rdb.Get(r.ctx, keyBridge+bridge.ID).Result(); err == nil {
		var oldBridge Bridge
		if err := json.Unmarshal([]byte(val), &oldBridge); err == nil {
			for _, channel := range oldBridge.Channels {
				if err := r.rdb.Del(r.ctx, keyBChannel+channel.ID).Err(); err != nil {
					return err
				}
			}
		}
	}

	if err := r.rdb.Set(r.ctx, keyBridge+bridge.ID, bridgeJSON, 0).Err(); err != nil {
		return err
	}

	for _, channel := range bridge.Channels {
		if err := r.rdb.Set(r.ctx, keyBChannel+channel.ID, bridge.ID, 0).Err(); err != nil {
			return err
		}
	}

	return nil
}

func (r redisDatabase) getBridge(id string) (Bridge, error) {
	val, err := r.rdb.Get(r.ctx, keyBridge+id).Result()
	if err != nil {
		if err == redis.Nil {
			return Bridge{}, nil
		}
		return Bridge{}, err
	}

	var bridge Bridge
	if err := json.Unmarshal([]byte(val), &bridge); err != nil {
		return Bridge{}, err
	}

	return bridge, nil
}

func (r redisDatabase) getBridgeByChannel(channelID string) (Bridge, error) {
	val, err := r.rdb.Get(r.ctx, keyBChannel+channelID).Result()
	if err != nil {
		if err == redis.Nil {
			return Bridge{}, nil
		}
		return Bridge{}, err
	}

	return r.getBridge(val)
}

func (r redisDatabase) createMessage(message BridgeMessageCollection) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if err := r.rdb.Set(r.ctx, keyMessage+message.ID, messageJSON, 0).Err(); err != nil {
		return err
	}

	for _, msg := range message.Messages {
		for _, id := range msg.ID {
			if err := r.rdb.Set(r.ctx, keyMessage+id, messageJSON, 0).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r redisDatabase) deleteMessage(id string) error {
	message, err := r.getMessage(id)
	if err != nil {
		return err
	}

	if err := r.rdb.Del(r.ctx, keyMessage+id).Err(); err != nil {
		return err
	}

	for _, msg := range message.Messages {
		for _, msgID := range msg.ID {
			if err := r.rdb.Del(r.ctx, keyMessage+msgID).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r redisDatabase) getMessage(id string) (BridgeMessageCollection, error) {
	val, err := r.rdb.Get(r.ctx, keyMessage+id).Result()
	if err != nil {
		if err == redis.Nil {
			return BridgeMessageCollection{}, nil
		}
		return BridgeMessageCollection{}, err
	}

	var message BridgeMessageCollection
	if err := json.Unmarshal([]byte(val), &message); err != nil {
		return BridgeMessageCollection{}, err
	}

	return message, nil
}

func (r redisDatabase) hasMessage(id string) bool {
	exists, err := r.rdb.Exists(r.ctx, keyMessage+id).Result()
	if err != nil {
		return false
	}
	return exists == 1
}

var ulidRegex = regexp.MustCompile("[0-7][0-9A-HJKMNP-TV-Z]{25}")
var uuidRegex = regexp.MustCompile("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")

func (r redisDatabase) GetAllBridges() ([]Bridge, error) {
	defer startSpinner().Stop()

	var cursor uint64
	bridges := make([]Bridge, 0)

	for {
		keys, nextCursor, err := r.rdb.Scan(r.ctx, cursor, keyBridge+"*", 10).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			val, err := r.rdb.Get(r.ctx, key).Result()
			if err != nil {
				return nil, err
			}
			var bridge Bridge

			if !r.svn {
				if err := json.Unmarshal([]byte(val), &bridge); err != nil {
					return nil, err
				}
			} else {
				if !ulidRegex.MatchString(strings.TrimPrefix(key, keyBridge)) && !uuidRegex.MatchString(strings.TrimPrefix(key, keyBridge)) {
					continue
				}

				var br struct {
					ID       string          `json:"id"`
					Channels []BridgeChannel `json:"channels"`
				}
				if err := json.Unmarshal([]byte(val), &br); err != nil {
					return nil, err
				}
				bridge = Bridge{
					ID:       strings.TrimPrefix(key, keyBridge),
					Channels: br.Channels,
					Name:     br.ID,
					Settings: BridgeSettings{false},
				}
			}

			bridges = append(bridges, bridge)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return bridges, nil
}

func (r redisDatabase) GetAllMessages() ([]BridgeMessageCollection, error) {
	defer startSpinner().Stop()

	var cursor uint64
	messages := make([]BridgeMessageCollection, 0)

	for {
		keys, nextCursor, err := r.rdb.Scan(r.ctx, cursor, keyMessage+"*", 10).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			val, err := r.rdb.Get(r.ctx, key).Result()
			if err != nil {
				return nil, err
			}

			var message BridgeMessageCollection
			if err := json.Unmarshal([]byte(val), &message); err != nil {
				return nil, err
			}

			messages = append(messages, message)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return messages, nil
}

func (r redisDatabase) SetAllBridges(bridges []Bridge) error {
	defer startSpinner().Stop()

	pipe := r.rdb.Pipeline()
	for _, bridge := range bridges {
		bridgeJSON, err := json.Marshal(bridge)
		if err != nil {
			return err
		}

		pipe.Set(r.ctx, keyBridge+bridge.ID, bridgeJSON, 0)
		for _, channel := range bridge.Channels {
			pipe.Set(r.ctx, keyBChannel+channel.ID, bridge.ID, 0)
		}
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

func (r redisDatabase) SetAllMessages(messages []BridgeMessageCollection) error {
	defer startSpinner().Stop()

	pipe := r.rdb.Pipeline()
	for _, message := range messages {
		messageJSON, err := json.Marshal(message)
		if err != nil {
			return err
		}

		pipe.Set(r.ctx, keyMessage+message.ID, messageJSON, 0)
		for _, msg := range message.Messages {
			for _, id := range msg.ID {
				pipe.Set(r.ctx, keyMessage+id, messageJSON, 0)
			}
		}
	}

	_, err := pipe.Exec(r.ctx)
	return err
}
