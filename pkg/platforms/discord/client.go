package discord

import (
	"errors"
	"sync"

	"codeberg.org/jersey/lightning/internal/cache"
)

type client struct {
	apiHost     string
	cdnHost     string
	product     string
	frontend    string
	version     string
	spacebar    bool
	token       string
	application *application
	intents     intent

	handlersMu sync.RWMutex
	handlers   map[eventType][]func(any)

	dms      cache.Expiring[string, *channel]
	channels cache.Expiring[string, *channel]
	guilds   cache.Expiring[snowflake, *guild]
	users    cache.Expiring[string, *user]
	members  cache.Expiring[snowflake, *member]
	roles    cache.Expiring[string, *role]
	emojis   cache.Expiring[snowflake, *[]discordEmoji]
	messages cache.Expiring[string, *message]
	webhooks cache.Expiring[snowflake, *webhook]
}

func (bot *client) bulkDelete(channel string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	if len(ids) == 1 {
		return bot.deleteMessage(channel, ids[0])
	}

	return bot.do("POST", "/channels/"+channel+"/messages/bulk-delete", map[string][]string{"messages": ids}, nil)
}

func (bot *client) createWebhook(channel, name string) (*webhook, error) {
	var res webhook

	err := bot.do("POST", "/channels/"+channel+"/webhooks", map[string]string{"name": name}, &res)
	if err != nil {
		return nil, err
	}

	bot.webhooks.Set(res.ID, &res)

	return &res, nil
}

func (bot *client) deleteMessage(channel, id string) error {
	err := bot.do("DELETE", "/channels/"+channel+"/messages/"+id, nil, nil)

	var aerr apiError
	if errors.As(err, &aerr) {
		if aerr.Code == 10008 {
			return nil
		}
	}

	return err
}

func (bot *client) editMessage(channel, id string, msg *messageEdit) error {
	var res message

	err := bot.do("PATCH", "/channels/"+channel+"/messages/"+id, msg, &res)
	if err != nil {
		var aerr apiError
		if errors.As(err, &aerr) {
			if aerr.Code == 10008 {
				return nil
			}
		}

		return err
	}

	bot.messages.Set(string(res.ID), &res)

	return nil
}

func (bot *client) editWebhook(
	webhook string, token string, id string, msg *webhookEditMessagePayload,
) error {
	var res message

	err := bot.do("PATCH", "/webhooks/"+webhook+"/"+token+"/messages/"+id, msg, &res)
	if err != nil {
		var aerr apiError
		if errors.As(err, &aerr) {
			if aerr.Code == 10008 {
				return nil
			}
		}

		return err
	}

	bot.messages.Set(string(res.ID), &res)

	return nil
}

func (bot *client) getChannel(channel string) (*channel, bool) {
	return getCached(bot, &bot.channels, "/channels/"+channel, channel)
}

func (bot *client) getEmojiByName(guild *snowflake, name string) (*discordEmoji, bool) {
	emojis, ok := getCached(bot, &bot.emojis, "/guilds/"+string(*guild)+"/emojis", *guild)
	if !ok {
		return nil, false
	}

	for _, emoji := range *emojis {
		if emoji.Name == name {
			return &emoji, true
		}
	}

	return nil, false
}

func (bot *client) getGuild(guild *snowflake) (*guild, bool) {
	if guild == nil {
		return nil, false
	}

	return getCached(bot, &bot.guilds, "/guilds/"+string(*guild), *guild)
}

func (bot *client) getMember(guild *snowflake, member snowflake) (*member, bool) {
	if guild == nil {
		return nil, false
	}

	return getCached(bot, &bot.members, "/guilds/"+string(*guild)+"/members/"+string(member), member)
}

func (bot *client) getMessage(channel, id string) (*message, bool) {
	return getCached(bot, &bot.messages, "/channels/"+channel+"/messages/"+id, id)
}

func (bot *client) getRole(guild *snowflake, role string) (*role, bool) {
	if guild == nil {
		return nil, false
	}

	return getCached(bot, &bot.roles, "/guilds/"+string(*guild)+"/roles/"+role, role)
}

func (bot *client) getUser(user string) (*user, bool) {
	return getCached(bot, &bot.users, "/users/"+user, user)
}

func (bot *client) getWebhook(webhook *snowflake) (*webhook, bool) {
	if webhook == nil {
		return nil, false
	}

	return getCached(bot, &bot.webhooks, "/webhooks/"+string(*webhook), *webhook)
}

func (bot *client) overwriteCommands(commands []applicationCommand) error {
	if bot.application == nil {
		if err := bot.do("GET", "/applications/@me", nil, &bot.application); err != nil {
			return err
		}
	}

	return bot.do("PUT", "/applications/"+string(bot.application.ID)+"/commands", commands, nil)
}

func (bot *client) respondInteraction(id snowflake, token string, msg *interactionResponse) error {
	return bot.doMultipart("POST", "/interactions/"+string(id)+"/"+token+"/callback", msg, msg.Data.Files, nil)
}

func (bot *client) sendMessage(channel string, msg *messageSend) (*message, error) {
	var res message

	if err := bot.doMultipart("POST", "/channels/"+channel+"/messages", msg, msg.Files, &res); err != nil {
		return nil, err
	}

	bot.messages.Set(string(res.ID), &res)

	return &res, nil
}

func (bot *client) sendWebhook(webhook, token string, msg *webhookExecutePayload) (*message, error) {
	var res message

	if err := bot.doMultipart("POST", "/webhooks/"+webhook+"/"+token+"?wait=true", msg, msg.Files, &res); err != nil {
		return nil, err
	}

	bot.messages.Set(string(res.ID), &res)

	return &res, nil
}

func (bot *client) startDM(user string) (*channel, error) {
	if dmChannel, ok := bot.dms.Get(user); ok {
		return dmChannel, nil
	}

	var dmChannel channel

	err := bot.do("POST", "/users/@me/channels", map[string]string{"recipient_id": user}, &dmChannel)
	if err != nil {
		return nil, err
	}

	bot.dms.Set(user, &dmChannel)
	bot.channels.Set(string(dmChannel.ID), &dmChannel)

	return &dmChannel, nil
}
