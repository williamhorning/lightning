package discord

import "github.com/bwmarrin/discordgo"

func setBaseURL(base string) {
	discordgo.EndpointDiscord = base
	discordgo.EndpointAPI = discordgo.EndpointDiscord + "api/v" + discordgo.APIVersion + "/"
	discordgo.EndpointGuilds = discordgo.EndpointAPI + "guilds/"
	discordgo.EndpointChannels = discordgo.EndpointAPI + "channels/"
	discordgo.EndpointUsers = discordgo.EndpointAPI + "users/"
	discordgo.EndpointGateway = discordgo.EndpointAPI + "gateway"
	discordgo.EndpointGatewayBot = discordgo.EndpointGateway + "/bot"
	discordgo.EndpointWebhooks = discordgo.EndpointAPI + "webhooks/"
	discordgo.EndpointStickers = discordgo.EndpointAPI + "stickers/"
	discordgo.EndpointStageInstances = discordgo.EndpointAPI + "stage-instances"
	discordgo.EndpointSKUs = discordgo.EndpointAPI + "skus"
	discordgo.EndpointVoice = discordgo.EndpointAPI + "/voice/"
	discordgo.EndpointVoiceRegions = discordgo.EndpointVoice + "regions"
	discordgo.EndpointNitroStickersPacks = discordgo.EndpointAPI + "/sticker-packs"
	discordgo.EndpointGuildCreate = discordgo.EndpointAPI + "guilds"
	discordgo.EndpointApplications = discordgo.EndpointAPI + "applications"
	discordgo.EndpointOAuth2 = discordgo.EndpointAPI + "oauth2/"
	discordgo.EndpointOAuth2Applications = discordgo.EndpointOAuth2 + "applications"
	discordgo.EndpointOauth2 = discordgo.EndpointOAuth2
	discordgo.EndpointOauth2Applications = discordgo.EndpointOAuth2Applications
}

func setCDNURL(cdn string) {
	discordgo.EndpointCDN = cdn
	discordgo.EndpointCDNAttachments = discordgo.EndpointCDN + "attachments/"
	discordgo.EndpointCDNAvatars = discordgo.EndpointCDN + "avatars/"
	discordgo.EndpointCDNIcons = discordgo.EndpointCDN + "icons/"
	discordgo.EndpointCDNSplashes = discordgo.EndpointCDN + "splashes/"
	discordgo.EndpointCDNChannelIcons = discordgo.EndpointCDN + "channel-icons/"
	discordgo.EndpointCDNBanners = discordgo.EndpointCDN + "banners/"
	discordgo.EndpointCDNGuilds = discordgo.EndpointCDN + "guilds/"
	discordgo.EndpointCDNRoleIcons = discordgo.EndpointCDN + "role-icons/"
}
