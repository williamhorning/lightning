package revolt

import (
	"time"

	"github.com/williamhorning/lightning/internal/cache"
)

type revoltCache struct {
	channelCache   *cache.Expiring[string, revoltChannel]
	dmChannelCache *cache.Expiring[string, revoltChannel]
	emojiCache     *cache.Expiring[string, revoltEmoji]
	memberCache    *cache.Expiring[string, revoltServerMember]
	serverCache    *cache.Expiring[string, revoltServer]
	userCache      *cache.Expiring[string, revoltUser]
}

func newRevoltCache() revoltCache {
	return revoltCache{
		cache.New[string, revoltChannel](30 * time.Second),
		cache.New[string, revoltChannel](30 * time.Second),
		cache.New[string, revoltEmoji](30 * time.Second),
		cache.New[string, revoltServerMember](30 * time.Second),
		cache.New[string, revoltServer](30 * time.Second),
		cache.New[string, revoltUser](30 * time.Second),
	}
}

func (p *revoltPlugin) setCache(ready *revoltEventReady) {
	for _, user := range ready.Users {
		p.userCache.Set(user.ID, *user)
	}

	for _, server := range ready.Servers {
		p.serverCache.Set(server.ID, *server)
	}

	for _, channel := range ready.Channels {
		p.channelCache.Set(channel.ID, *channel)
	}

	for _, member := range ready.Members {
		p.memberCache.Set(member.ID.Server+"-"+member.ID.User, *member)
	}

	for _, emoji := range ready.Emojis {
		p.emojiCache.Set(emoji.ID, *emoji)
	}
}
