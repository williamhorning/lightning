package revolt

import "github.com/williamhorning/lightning/internal/cache"

type revoltCache struct {
	channelCache   *cache.Expiring[string, revoltChannel]
	dmChannelCache *cache.Expiring[string, revoltChannel]
	emojiCache     *cache.Expiring[string, revoltEmoji]
	emojiNameCache *cache.Expiring[string, revoltEmoji]
	memberCache    *cache.Expiring[string, revoltServerMember]
	serverCache    *cache.Expiring[string, revoltServer]
	userCache      *cache.Expiring[string, revoltUser]
}

func newRevoltCache() revoltCache {
	return revoltCache{
		cache.New[string, revoltChannel](cache.DefaultTTL),
		cache.New[string, revoltChannel](cache.DefaultTTL),
		cache.New[string, revoltEmoji](cache.DefaultTTL),
		cache.New[string, revoltEmoji](cache.DefaultTTL),
		cache.New[string, revoltServerMember](cache.DefaultTTL),
		cache.New[string, revoltServer](cache.DefaultTTL),
		cache.New[string, revoltUser](cache.DefaultTTL),
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

		if emoji.Parent != nil {
			p.emojiNameCache.Set(emoji.Parent.ID+"-"+emoji.Name, *emoji)
		}
	}
}
