package revolt

import "time"

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
		return nil, handleDMChannel(channelData)
	case revoltChannelTypeText, revoltChannelTypeVoice:
		return nil, p.handleTextOrVoiceChannel(channelData)
	default:
		return nil, &revoltPermissionsError{"unknown channel type", 0, 0}
	}
}

func handleDMChannel(channel *revoltChannel) error {
	if channel.Permissions == nil || *channel.Permissions&messageSendPermission != messageSendPermission {
		return &revoltPermissionsError{"DM", *channel.Permissions, messageSendPermission}
	}

	return nil
}

func (p *revoltPlugin) handleTextOrVoiceChannel(channel *revoltChannel) error {
	server := p.getServer(channel.Server)
	if server == nil {
		return &revoltPermissionsError{"nil server (" + channel.Server + ")", 0, correctPermissionValue}
	}

	if server.Owner == p.self.ID {
		return nil
	}

	member := p.getMember(channel.Server, p.self.ID)
	if member == nil {
		return &revoltPermissionsError{"server (with nil bot member)", 0, correctPermissionValue}
	}

	if member.Timeout != nil && time.Now().Before(*member.Timeout) {
		return &revoltPermissionsError{"server (with bot in timeout)", 0, correctPermissionValue}
	}

	permissions := calculatePermissions(server, member, channel)

	if (permissions & correctPermissionValue) != correctPermissionValue {
		return &revoltPermissionsError{"server channel (with missing permissions)", permissions, correctPermissionValue}
	}

	return nil
}

func calculatePermissions(svr *revoltServer, member *revoltServerMember, channel *revoltChannel) uint {
	permissions := *svr.DefaultPermissions

	for _, roleID := range member.Roles {
		role, ok := svr.Roles[roleID]
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
