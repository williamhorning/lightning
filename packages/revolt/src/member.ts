import type { Channel, Member } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';

const member_cache = new Map<`${string}/${string}`, {
	value: Member;
	expiry: number;
}>();

export async function fetch_member(
	client: Client,
	channel: Channel & { channel_type: 'TextChannel' },
	user: string,
): Promise<Member> {
	const time_now = Temporal.Now.instant().epochMilliseconds;

	const member = member_cache.get(`${channel.server}/${user}`);

	if (member && member.expiry > time_now) {
		return member.value;
	}

	const response = await client.request(
		'get',
		`/servers/${channel.server}/members/${user}`,
		undefined,
	) as Member;

	member_cache.set(`${channel.server}/${user}`, {
		value: response,
		expiry: time_now + 300000,
	});

	return response;
}
