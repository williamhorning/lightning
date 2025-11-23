package stoat

import (
	"slices"
	"time"
)

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
	permissions := server.DefaultPermissions

	for _, roleID := range member.Roles {
		role, ok := server.Roles[roleID]
		if ok {
			permissions |= role.Permissions.Allow
			permissions &= ^role.Permissions.Deny
		}
	}

	if member.Timeout.After(time.Now()) {
		permissions &= PermissionSet3
	}

	if channel.DefaultPerms != nil {
		permissions |= channel.DefaultPerms.Allow
		permissions &= ^channel.DefaultPerms.Deny
	}

	for id, role := range channel.RolePermissions {
		if slices.Contains(member.Roles, id) {
			permissions |= role.Allow
			permissions &= ^role.Deny
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
	PermissionManageChannel       Permission = 1 << iota // Manage the channel or channels on the server
	PermissionManageServer                               // Manage the server
	PermissionManagePermissions                          // Manage permissions on servers or channels
	PermissionManageRole                                 // Manage roles on server
	PermissionManageCustomization                        // Manage emoji on servers
	PermissionKickMembers                                // Kick other members below their ranking
	PermissionBanMembers                                 // Ban other members below their ranking
	PermissionTimeoutMembers                             // Timeout other members below their ranking
	PermissionAssignRoles                                // Assign roles to members below their ranking
	PermissionChangeNickname                             // Change own nickname
	PermissionManageNicknames                            // Change or remove other's nicknames below their ranking
	PermissionChangeAvatar                               // Change own avatar
	PermissionRemoveAvatars                              // Remove other's avatars below their ranking
	PermissionViewChannel                                // View a channel
	PermissionReadMessageHistory                         // Read a channel's past message history
	PermissionSendMessage                                // Send a message in a channel
	PermissionManageMessages                             // Delete messages in a channel
	PermissionManageWebhooks                             // Manage webhook entries on a channel
	PermissionInviteOthers                               // Create invites to this channel
	PermissionSendEmbeds                                 // Send embedded content in this channel
	PermissionUploadFiles                                // Send attachments and media in this channel
	PermissionMasquerade                                 // Masquerade messages using custom nickname and avatar
	PermissionReact                                      // React to messages with emojis
	PermissionConnect                                    // Connect to a voice channel
	PermissionSpeak                                      // Speak in a voice call
	PermissionVideo                                      // Share video in a voice call
	PermissionMuteMembers                                // Mute other members with lower ranking in a voice call
	PermissionDeafenMembers                              // Deafen other members with lower ranking in a voice call
	PermissionMoveMembers                                // Move members between voice channels
	PermissionAll                 Permission = 0x000F_FFFF_FFFF_FFFF
	PermissionSet1                Permission = PermissionSet3 | PermissionSendMessage |
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
