package rvapi

import (
	"slices"
	"time"
)

// GetPermissions returns the permissions for the user in the given channel.
func (s *Session) GetPermissions(user *User, channel *Channel) Permission {
	switch channel.ChannelType {
	case ChannelTypeDM:
		return calculateUserPermissions(s, user, channel)
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
		return calculateServerPermissions(s, channel, user)
	default:
		return 0
	}
}

func calculateUserPermissions(session *Session, self *User, channel *Channel) Permission {
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

	recipient := session.User(userID)
	if recipient == nil {
		return PermissionSet3
	}

	if recipient.Relationship == RelationshipFriend || recipient.Relationship == RelationshipUser {
		return PermissionSet1
	}

	return PermissionSet3
}

func calculateServerPermissions(session *Session, channel *Channel, user *User) Permission {
	server := session.Server(*channel.Server)
	if server == nil {
		return 0
	}

	if server.Owner == user.ID {
		return PermissionAll
	}

	member := session.Member(*channel.Server, user.ID)
	if member == nil {
		return 0
	}

	return getMemberPermissions(member, server, server.DefaultPermissions, channel)
}

func getMemberPermissions(member *Member, server *Server, permissions Permission, channel *Channel) Permission {
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
