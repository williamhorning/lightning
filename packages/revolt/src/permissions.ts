import type { Client } from '@jersey/rvapi';
import { log_error } from '@lightning/lightning';
import {
	fetch_channel,
	fetch_member,
	fetch_role,
	fetch_server,
} from './cache.ts';
import { handle_error } from './errors.ts';

const needed_permissions = 485495808;
const error_message = 'missing ChangeNickname, ChangeAvatar, ReadMessageHistory, \
SendMessage, ManageMessages, SendEmbeds, UploadFiles, and/or Masquerade permissions';

export async function check_permissions(
	channel_id: string,
	bot_id: string,
	client: Client,
) {
	try {
		const channel = await fetch_channel(client, channel_id);

		if (channel.channel_type === 'Group') {
			if (
				!(channel.permissions && (channel.permissions & needed_permissions))
			) log_error(error_message);
		} else if (channel.channel_type === 'TextChannel') {
			const server = await fetch_server(client, channel.server);
			const member = await fetch_member(client, channel.server, bot_id);

			// check server permissions
			let currentPermissions = server.default_permissions;

			for (const role of (member.roles || [])) {
				const { permissions: role_permissions } = await fetch_role(
					client,
					server._id,
					role,
				);

				currentPermissions |= role_permissions.a || 0;
				currentPermissions &= ~role_permissions.d || 0;
			}

			// apply default allow/denies
			if (channel.default_permissions) {
				currentPermissions |= channel.default_permissions.a;
				currentPermissions &= ~channel.default_permissions.d;
			}

			// apply role permissions
			if (channel.role_permissions) {
				for (const role of (member.roles || [])) {
					currentPermissions |= channel.role_permissions[role]?.a || 0;
					currentPermissions &= ~channel.role_permissions[role]?.d || 0;
				}
			}

			if (!(currentPermissions & needed_permissions)) log_error(error_message);
		} else {
			log_error(`unsupported channel type: ${channel.channel_type}`);
		}

		return channel_id;
	} catch (e) {
		handle_error(e);
	}
}
