import type { Channel, Member } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';

interface member_map_value {
	value: Member;
	expiry: number;
}

const member_map = new Map<string, member_map_value>();

export async function fetch_member(
	client: Client,
	channel: Channel & { channel_type: 'TextChannel' },
	user_id: string,
): Promise<Member> {
	const time_now = Temporal.Now.instant().epochMilliseconds;

	const member = member_map.get(user_id);

	if (member && member.expiry > time_now) {
		return member.value;
	}

	const member_resp = await client.request(
		'get',
		`/servers/${channel.server}/members/${user_id}`,
		undefined,
	);

	member_map.set(user_id, {
		value: member_resp as Member,
		expiry: time_now + 300000,
	});

	return member_resp as Member;
}
