package stoat

import "time"

func (session *session) getPermissions(user *stUser, channel *stChannel) stPermission {
	switch channel.ChannelType {
	case "DirectMessage":
		return session.calculateUserPermissions(user, channel)
	case "Group":
		if channel.Owner != nil && *channel.Owner == user.ID {
			return stPermissionAll
		}

		if channel.Permissions == nil {
			return stPermissionSet1
		}

		return *channel.Permissions
	case "SavedMessages":
		return stPermissionAll
	case "TextChannel", "VoiceChannel":
		return session.calculateServerPermissions(channel, user)
	default:
		return 0
	}
}

func (session *session) calculateUserPermissions(self *stUser, channel *stChannel) stPermission {
	userID := ""

	for _, recipient := range channel.Recipients {
		if recipient != self.ID {
			userID = recipient

			break
		}
	}

	if userID == "" {
		return stPermissionSet3
	}

	recipient, err := get(session, "/users/"+userID, userID, &session.userCache)
	if err != nil {
		return stPermissionSet3
	}

	if recipient.Relationship == "Friend" || recipient.Relationship == "User" {
		return stPermissionSet1
	}

	return stPermissionSet3
}

func (session *session) calculateServerPermissions(channel *stChannel, user *stUser) stPermission {
	if channel.Server == nil {
		return 0
	}

	server, err := get(session, "/servers/"+*channel.Server, *channel.Server, &session.serverCache)
	if err != nil {
		return 0
	}

	if server.Owner == user.ID {
		return stPermissionAll
	}

	member, err := get(session, "/servers/"+server.ID+"/members/"+user.ID, server.ID+"-"+user.ID, &session.memberCache)
	if err != nil {
		return 0
	}

	return getMemberPermissions(member, server, channel)
}

func getMemberPermissions(member *stMember, server *stServer, channel *stChannel) stPermission {
	if server.Owner == member.ID.User {
		return stPermissionAll
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
		permissions &= stPermissionSet3
	}

	return permissions
}

// stPermission type.
type stPermission uint64

// Individual permission flags.
const (
	stPermissionManageChannel stPermission = 1 << iota
	stPermissionManageServer
	stPermissionManagePermissions
	stPermissionManageRole
	stPermissionManageCustomization
	stPermissionKickMembers stPermission = 1 << (iota + 1)
	stPermissionBanMembers
	stPermissionTimeoutMembers
	stPermissionAssignRoles
	stPermissionChangeNickname
	stPermissionManageNicknames
	stPermissionChangeAvatar
	stPermissionRemoveAvatars
	stPermissionViewChannel stPermission = 1 << (iota + 7)
	stPermissionReadMessageHistory
	stPermissionSendMessage
	stPermissionManageMessages
	stPermissionManageWebhooks
	stPermissionInviteOthers
	stPermissionSendEmbeds
	stPermissionUploadFiles
	stPermissionMasquerade
	stPermissionReact
	stPermissionConnect
	stPermissionSpeak
	stPermissionVideo
	stPermissionMuteMembers
	stPermissionDeafenMembers
	stPermissionMoveMembers
	stPermissionListen
	stPermissionMentionEveryone
	stPermissionMentionRoles

	stPermissionAll  stPermission = 0x000F_FFFF_FFFF_FFFF
	stPermissionSet1 stPermission = stPermissionSet3 | stPermissionSendMessage |
		stPermissionManageChannel | stPermissionConnect | stPermissionSendEmbeds | stPermissionInviteOthers |
		stPermissionUploadFiles
	stPermissionSet2 stPermission = stPermissionSet1 | stPermissionChangeNickname | stPermissionChangeAvatar
	stPermissionSet3 stPermission = stPermissionViewChannel | stPermissionReadMessageHistory
)

// stPermissionNames is a map of individual permission names.
var stPermissionNames = map[stPermission]string{ //nolint:gochecknoglobals
	stPermissionManageChannel:       "Manage Channel",
	stPermissionManageServer:        "Manage Server",
	stPermissionManagePermissions:   "Manage Permissions",
	stPermissionManageRole:          "Manage Role",
	stPermissionManageCustomization: "Manage Customization",
	stPermissionKickMembers:         "Kick Members",
	stPermissionBanMembers:          "Ban Members",
	stPermissionTimeoutMembers:      "Timeout Members",
	stPermissionAssignRoles:         "Assign Roles",
	stPermissionChangeNickname:      "Change Nickname",
	stPermissionManageNicknames:     "Manage Nicknames",
	stPermissionChangeAvatar:        "Change Avatar",
	stPermissionRemoveAvatars:       "Remove Avatars",
	stPermissionViewChannel:         "View Channel",
	stPermissionReadMessageHistory:  "Read Message History",
	stPermissionSendMessage:         "Send Message",
	stPermissionManageMessages:      "Manage Messages",
	stPermissionManageWebhooks:      "Manage Webhooks",
	stPermissionInviteOthers:        "Invite Others",
	stPermissionSendEmbeds:          "Send Embeds",
	stPermissionUploadFiles:         "Upload Files",
	stPermissionMasquerade:          "Masquerade",
	stPermissionReact:               "React",
	stPermissionConnect:             "Connect",
	stPermissionSpeak:               "Speak",
	stPermissionVideo:               "Video",
	stPermissionMuteMembers:         "Mute Members",
	stPermissionDeafenMembers:       "Deafen Members",
	stPermissionMoveMembers:         "Move Members",
}
