package rvapi

// Channel gets a channel.
func (session *Session) Channel(chID string) *Channel {
	if channel, ok := session.ChannelCache.Get(chID); ok {
		return &channel
	}

	var channel Channel

	if Get(session, "/channels/"+chID, &channel) != nil {
		return nil
	}

	return &channel
}

// Member gets a member.
func (session *Session) Member(server, user string) *Member {
	if member, ok := session.MemberCache.Get(server + "-" + user); ok {
		return &member
	}

	var member Member

	if Get(session, "/servers/"+server+"/members/"+user, &member) != nil {
		return nil
	}

	return &member
}

// User gets a user.
func (session *Session) User(userID string) *User {
	if user, ok := session.UserCache.Get(userID); ok {
		return &user
	}

	var user User

	if Get(session, "/users/"+userID, &user) != nil {
		return nil
	}

	return &user
}

// ServerEmoji gets emoji for a server.
func (session *Session) ServerEmoji(server string) []Emoji {
	if emoji, ok := session.ServerEmojiCache.Get(server); ok {
		return emoji
	}

	var emoji []Emoji

	if Get(session, "/servers/"+server+"/emojis", &emoji) != nil {
		return nil
	}

	return emoji
}

// Emoji gets emoji.
func (session *Session) Emoji(emojiID string) *Emoji {
	if emoji, ok := session.EmojiCache.Get(emojiID); ok {
		return &emoji
	}

	var emoji Emoji

	if Get(session, "/custom/emoji/"+emojiID, &emoji) != nil {
		return nil
	}

	return &emoji
}

// Server gets server.
func (session *Session) Server(svr string) *Server {
	if server, ok := session.ServerCache.Get(svr); ok {
		return &server
	}

	var server Server

	if Get(session, "/custom/emoji/"+svr, &server) != nil {
		return nil
	}

	return &server
}
