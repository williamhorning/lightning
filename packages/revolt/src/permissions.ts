import type { Client } from '@jersey/rvapi';
import { LightningError, log_error } from '@jersey/lightning';
import { handle_error } from './errors.ts';
import { fetchChannel, fetchMember, fetchRole, fetchServer } from './cache.ts';

const permission_bits = [
	1 << 23, // ManageMessages
	1 << 28, // Masquerade
];

const needed_permissions = permission_bits.reduce((a, b) => a | b, 0);

export async function check_permissions(
	channelID: string,
	botID: string,
	client: Client,
) {
	try {
		const channel = await fetchChannel(client, channelID);

		if (channel.channel_type === 'Group') {
			if (channel.permissions && (channel.permissions & needed_permissions)) {
				return channel._id;
			}

			log_error('missing ManageMessages and/or Masquerade permission');
		} else if (channel.channel_type === 'TextChannel') {
			const server = await fetchServer(client, channel.server);
			const member = await fetchMember(client, channel.server, botID);

			// check server permissions
			let currentPermissions = server.default_permissions;

			for (const role of (member.roles || [])) {
				const { permissions: role_permissions } = await fetchRole(
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

			if (currentPermissions & needed_permissions) return channel._id;

			log_error('missing ManageMessages and/or Masquerade permission');
		} else {
			log_error(`unsupported channel type: ${channel.channel_type}`);
		}
	} catch (e) {
		if (e instanceof LightningError) throw e;

		handle_error(e);
	}
}
