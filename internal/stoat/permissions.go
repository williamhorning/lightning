package stoat

import "time"

// GetPermissions returns the permissions for the user in the given channel.
func (session *Session) GetPermissions(user *User, channel *Channel) Permission {
	switch channel.ChannelType {
	case ChannelTypeDM:
		return session.calculateUserPermissions(user, channel)
	case ChannelTypeGroup:
		if channel.Owner != nil && *channel.Owner == user.ID {
			return PermissionAll
		}

		if channel.Permissions == nil {
			return PermissionSet1
		}

		return *channel.Permissions
	case ChannelTypeSavedMessages:
		return PermissionAll
	case ChannelTypeText, ChannelTypeVoice:
		return session.calculateServerPermissions(channel, user)
	default:
		return 0
	}
}

func (session *Session) calculateUserPermissions(self *User, channel *Channel) Permission {
	userID := ""

	for _, recipient := range channel.Recipients {
		if recipient != self.ID {
			userID = recipient

			break
		}
	}

	if userID == "" {
		return PermissionSet3
	}

	recipient, err := Get(session, "/users/"+userID, userID, &session.UserCache)
	if err != nil {
		return PermissionSet3
	}

	if recipient.Relationship == RelationshipFriend || recipient.Relationship == RelationshipUser {
		return PermissionSet1
	}

	return PermissionSet3
}

func (session *Session) calculateServerPermissions(channel *Channel, user *User) Permission {
	if channel.Server == nil {
		return 0
	}

	server, err := Get(session, "/servers/"+*channel.Server, *channel.Server, &session.ServerCache)
	if err != nil {
		return 0
	}

	if server.Owner == user.ID {
		return PermissionAll
	}

	member, err := Get(session, "/servers/"+server.ID+"/members/"+user.ID, server.ID+"-"+user.ID, &session.MemberCache)
	if err != nil {
		return 0
	}

	return getMemberPermissions(member, server, channel)
}

func getMemberPermissions(member *Member, server *Server, channel *Channel) Permission {
	if server.Owner == member.ID.User {
		return PermissionAll
	}

	permissions := server.DefaultPermissions

	for _, roleID := range member.Roles {
		role, ok := server.Roles[roleID]
		if ok {
			permissions |= role.Permissions.Allow
			permissions &^= role.Permissions.Deny
		}
	}

	if channel.DefaultPerms != nil {
		permissions |= channel.DefaultPerms.Allow
		permissions &^= channel.DefaultPerms.Deny
	}

	for _, roleID := range member.Roles {
		role, ok := channel.RolePermissions[roleID]
		if ok {
			permissions |= role.Allow
			permissions &^= role.Deny
		}
	}

	if member.Timeout.After(time.Now()) {
		permissions &= PermissionSet3
	}

	return permissions
}

// Permission type.
type Permission uint64

// Individual permission flags.
const (
	PermissionManageChannel Permission = 1 << iota
	PermissionManageServer
	PermissionManagePermissions
	PermissionManageRole
	PermissionManageCustomization
	PermissionKickMembers Permission = 1 << (iota + 1)
	PermissionBanMembers
	PermissionTimeoutMembers
	PermissionAssignRoles
	PermissionChangeNickname
	PermissionManageNicknames
	PermissionChangeAvatar
	PermissionRemoveAvatars
	PermissionViewChannel Permission = 1 << (iota + 7)
	PermissionReadMessageHistory
	PermissionSendMessage
	PermissionManageMessages
	PermissionManageWebhooks
	PermissionInviteOthers
	PermissionSendEmbeds
	PermissionUploadFiles
	PermissionMasquerade
	PermissionReact
	PermissionConnect
	PermissionSpeak
	PermissionVideo
	PermissionMuteMembers
	PermissionDeafenMembers
	PermissionMoveMembers
	PermissionListen
	PermissionMentionEveryone
	PermissionMentionRoles

	PermissionAll  Permission = 0x000F_FFFF_FFFF_FFFF
	PermissionSet1 Permission = PermissionSet3 | PermissionSendMessage |
		PermissionManageChannel | PermissionConnect | PermissionSendEmbeds | PermissionInviteOthers |
		PermissionUploadFiles
	PermissionSet2 Permission = PermissionSet1 | PermissionChangeNickname | PermissionChangeAvatar
	PermissionSet3 Permission = PermissionViewChannel | PermissionReadMessageHistory
)

// PermissionNames is a map of individual permission names.
var PermissionNames = map[Permission]string{ //nolint:gochecknoglobals
	PermissionManageChannel:       "Manage Channel",
	PermissionManageServer:        "Manage Server",
	PermissionManagePermissions:   "Manage Permissions",
	PermissionManageRole:          "Manage Role",
	PermissionManageCustomization: "Manage Customization",
	PermissionKickMembers:         "Kick Members",
	PermissionBanMembers:          "Ban Members",
	PermissionTimeoutMembers:      "Timeout Members",
	PermissionAssignRoles:         "Assign Roles",
	PermissionChangeNickname:      "Change Nickname",
	PermissionManageNicknames:     "Manage Nicknames",
	PermissionChangeAvatar:        "Change Avatar",
	PermissionRemoveAvatars:       "Remove Avatars",
	PermissionViewChannel:         "View Channel",
	PermissionReadMessageHistory:  "Read Message History",
	PermissionSendMessage:         "Send Message",
	PermissionManageMessages:      "Manage Messages",
	PermissionManageWebhooks:      "Manage Webhooks",
	PermissionInviteOthers:        "Invite Others",
	PermissionSendEmbeds:          "Send Embeds",
	PermissionUploadFiles:         "Upload Files",
	PermissionMasquerade:          "Masquerade",
	PermissionReact:               "React",
	PermissionConnect:             "Connect",
	PermissionSpeak:               "Speak",
	PermissionVideo:               "Video",
	PermissionMuteMembers:         "Mute Members",
	PermissionDeafenMembers:       "Deafen Members",
	PermissionMoveMembers:         "Move Members",
}
