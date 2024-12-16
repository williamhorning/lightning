import type { Client } from '@jersey/rvapi';
import type { Channel, Member, Role, Server } from '@jersey/revolt-api-types';
import { LightningError, log_error } from '@jersey/lightning';
import { handle_error } from './error_handler.ts';

const permissions_to_check = [
    1 << 23, // ManageMessages
    1 << 28, // Masquerade
];

const permissions = permissions_to_check.reduce((a, b) => a | b, 0);

export async function check_permissions(channel_id: string, client: Client, bot_id: string) {
	try {
		const channel = await client.request(
			'get',
			`/channels/${channel_id}`,
			undefined,
		) as Channel;

        if (channel.channel_type === 'Group') {
            if (channel.permissions && (channel.permissions & permissions)) return channel;

            log_error('insufficient group permissions: missing ManageMessages and/or Masquerade');
        } else if (channel.channel_type === 'TextChannel') {
            return await server_permissions(channel, client, bot_id);
        } else {
            log_error(`unsupported channel type: ${channel.channel_type}`)
        }

	} catch (e) {
        if (e instanceof LightningError) throw e;

		handle_error(e);
	}
}

async function server_permissions(channel: Channel, client: Client, bot_id: string) {
    const server = await client.request(
        'get',
        `/servers/${channel.server}`,
        undefined,
    ) as Server;

    const member = await client.request(
        'get',
        `/servers/${channel.server}/members/${bot_id}`,
        undefined,
    ) as Member;

    // check server permissions
    let total_permissions = server.default_permissions;

    for (const role of (member.roles || [])) {
        const { permissions: role_perms } = await client.request(
            'get',
            `/servers/${channel.server}/roles/${role}`,
            undefined,
        ) as Role;

        total_permissions |= role_perms.a || 0;
        total_permissions &= ~role_perms.d || 0;
    }

    // apply default allow/denies
    if (channel.default_permissions) {
        total_permissions |= channel.default_permissions.a;
        total_permissions &= ~channel.default_permissions.d;
    }

    // apply role permissions
    if (channel.role_permissions) {
        for (const role of (member.roles || [])) {
            total_permissions |= channel.role_permissions[role]?.a || 0;
            total_permissions &= ~channel.role_permissions[role]?.d || 0;
        }
    }

    if (total_permissions & permissions) return channel;

    log_error('insufficient group permissions: missing ManageMessages and/or Masquerade');
}