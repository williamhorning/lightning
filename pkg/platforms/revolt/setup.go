package revolt

import (
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	correctPermissionValue = uint(485495808)
	messageSendPermission  = uint(2 << 22)
)

func (p *revoltPlugin) SetupChannel(channel string) (any, error) {
	channelData := p.getChannel(channel)

	switch channelData.ChannelType {
	case revoltChannelTypeSavedMessages:
		return channel, nil
	case revoltChannelTypeDM:
		return handleDMChannel(channelData)
	case revoltChannelTypeGroup:
		return p.handleGroupChannel(channelData)
	case revoltChannelTypeText, revoltChannelTypeVoice:
		return p.handleTextOrVoiceChannel(channelData)
	default:
		return nil, lightning.LogError(revoltPermissionsError{}, "Unknown channel type", nil, nil)
	}
}

func handleDMChannel(channel *revoltChannel) (any, error) {
	if channel.Permissions == nil || *channel.Permissions&messageSendPermission != messageSendPermission {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"DM permissions on Revolt are incorrect",
			map[string]any{"permissions": channel.Permissions},
			nil,
		)
	}

	return channel.ID, nil
}

func (p *revoltPlugin) handleGroupChannel(channel *revoltChannel) (any, error) {
	if channel.Owner == p.self.ID {
		return channel.ID, nil
	}

	if channel.Permissions == nil {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"Group permissions on Revolt are nil",
			nil,
			nil,
		)
	}

	return channel.ID, nil
}

func (p *revoltPlugin) handleTextOrVoiceChannel(channel *revoltChannel) (any, error) {
	server := p.getServer(channel.Server)
	if server == nil {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"Can't get server permissions on Revolt: server is nil",
			nil,
			nil,
		)
	}

	if server.Owner == p.self.ID {
		return channel.ID, nil
	}

	member := p.getMember(channel.Server, p.self.ID)
	if member == nil {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"Can't get server permissions on Revolt: bot member is nil",
			nil,
			nil,
		)
	}

	if member.Timeout != nil && time.Now().Before(*member.Timeout) {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"Can't setup this channel, I'm in timeout!",
			nil,
			nil,
		)
	}

	permissions := calculatePermissions(server, member, channel)

	slog.Debug("revolt: permissions", "channel", channel.ID, "permissions", permissions)

	if (permissions & correctPermissionValue) != correctPermissionValue {
		return nil, lightning.LogError(
			revoltPermissionsError{},
			"Missing permissions. Please add permissions to a role, assign that role to the bot, and rejoin the bridge",
			map[string]any{
				"channel":              channel.ID,
				"current_permissions":  permissions,
				"expected_permissions": correctPermissionValue,
			},
			nil,
		)
	}

	return channel.ID, nil
}

func calculatePermissions(
	server *revoltServer,
	member *revoltServerMember,
	channel *revoltChannel,
) uint {
	permissions := *server.DefaultPermissions

	for _, roleID := range member.Roles {
		role, ok := server.Roles[roleID]
		if ok {
			permissions |= role.Permissions.Allow
			permissions &= ^role.Permissions.Deny
		}
	}

	if channel.DefaultPermissions != nil {
		permissions |= channel.DefaultPermissions.Allow
		permissions &= ^channel.DefaultPermissions.Deny
	}

	return permissions
}
