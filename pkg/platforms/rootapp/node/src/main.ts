import { env } from "node:process";

import {
	rootServer,
	ChannelMessageEvent,
	type CommunityMember,
	type ChannelMessage,
} from "@rootsdk/server-bot";

import { Attachment, type DeletedMessage, type Message } from "./types";

rootServer.lifecycle.start(async () => {
	rootServer.community.channelMessages.on(
		ChannelMessageEvent.ChannelMessageCreated,
		getLightningMessageHandler("create"),
	);

	rootServer.community.channelMessages.on(
		ChannelMessageEvent.ChannelMessageEdited,
		getLightningMessageHandler("edit"),
	);

	rootServer.community.channelMessages.on(
		ChannelMessageEvent.ChannelMessageDeleted,
		async (evt) => {
			await fetch(env.LIGHTNING_WEBHOOK_URL + "/delete", {
				method: "POST",
				body: JSON.stringify({
					ChannelID: evt.channelId,
					EventID: evt.id,
					Time: evt.deletedAt,
				} as DeletedMessage),
			});
		},
	);
});

function getLightningMessageHandler(
	type: "create" | "edit",
): (evt: ChannelMessage) => Promise<void> {
	return async (evt: ChannelMessage) => {
		let user: CommunityMember;

		try {
			user = await rootServer.community.communityMembers.get({
				userId: evt.userId,
			});
		} catch {
			user = {
				nickname: "Root User",
				userId: evt.userId,
			};
		}

		let msg: Message = {
			Attachments: getAttachments(evt),
			Author: {
				ID: user.userId,
				Nickname: user.nickname,
				Username: user.nickname,
				Color: user.roleColorHex,
				ProfilePicture: user.profilePictureAssetUri,
			},
			ChannelID: evt.channelId,
			Content: evt.messageContent,
			Embeds: [], // not supported
			Emoji: [], // not supported
			EventID: evt.id,
			RepliedTo: [], // not supported
			Time: new Date(),
		};

		await fetch(env.LIGHTNING_WEBHOOK_URL + "/" + type, {
			method: "POST",
			body: JSON.stringify(msg),
		});
	};
}
function getAttachments(evt: ChannelMessage): Attachment[] {
	return (evt.messageUris ?? [])
		.map((val) => {
			if (!val.attachment || !evt.referenceMaps?.assets) return [];

			let item = evt.referenceMaps.assets[val.uri].link;
			let url: string;

			switch (item.oneofKind) {
				case "image":
					url = item.image.assetLinks[0].url;

					break;
				case "video":
					url = item.video.downloadUrl;

					break;
				case "url":
					url = item.url;

					break;
				case "invalid":
					return [];
				default:
					return [];
			}

			return [
				{
					Name: val.attachment.fileName,
					URL: url,
					Size: val.attachment.length,
				},
			];
		})
		.flat();
}
