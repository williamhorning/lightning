package revolt

import (
	"log/slog"
	"time"
)

const (
	correctPermissionValue = uint(481301504)
	messageSendPermission  = uint(2 << 22)
)

func (p *revoltPlugin) SetupChannel(channel string) (any, error) {
	channelData := p.getChannel(channel)

	switch channelData.ChannelType {
	case revoltChannelTypeSavedMessages, revoltChannelTypeGroup:
		return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
	case revoltChannelTypeDM:
		return handleDMChannel(channelData)
	case revoltChannelTypeText, revoltChannelTypeVoice:
		return p.handleTextOrVoiceChannel(channelData)
	default:
		return nil, &revoltPermissionsError{"unknown channel type"}
	}
}

func handleDMChannel(channel *revoltChannel) (any, error) {
	if channel.Permissions == nil || *channel.Permissions&messageSendPermission != messageSendPermission {
		slog.Error("revolt: insufficient permissions for DM channel", "permissions", channel.Permissions)

		return nil, &revoltPermissionsError{"DM"}
	}

	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *revoltPlugin) handleTextOrVoiceChannel(channel *revoltChannel) (any, error) {
	server := p.getServer(channel.Server)
	if server == nil {
		return nil, &revoltPermissionsError{"nil server (" + channel.Server + ")"}
	}

	if server.Owner == p.self.ID {
		return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
	}

	member := p.getMember(channel.Server, p.self.ID)
	if member == nil {
		slog.Error("revolt: bot member is nil", "server", channel.Server)

		return nil, &revoltPermissionsError{"server (with nil bot member)"}
	}

	if member.Timeout != nil && time.Now().Before(*member.Timeout) {
		slog.Error("revolt: bot is in timeout", "server", channel.Server, "timeout", member.Timeout)

		return nil, &revoltPermissionsError{"server (with bot in timeout)"}
	}

	permissions := calculatePermissions(server, member, channel)

	slog.Debug("revolt: permissions", "channel", channel.ID, "permissions", permissions)

	if (permissions & correctPermissionValue) != correctPermissionValue {
		slog.Error("revolt: insufficient permissions", "channel", channel.ID, "current_permissions", permissions,
			"expected_permissions", correctPermissionValue)

		return nil, &revoltPermissionsError{"server channel (with missing permissions)"}
	}

	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
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
