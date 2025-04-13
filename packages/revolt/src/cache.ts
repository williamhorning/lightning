import type { message_author } from '@jersey/lightning';
import type {
	Channel,
	Masquerade,
	Member,
	Message,
	Role,
	Server,
	User,
} from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';

class cacher<K extends string, V> {
	private map = new Map<K, {
		value: V;
		expiry: number;
	}>();
	get(key: K): V | undefined {
		const time = Temporal.Now.instant().epochMilliseconds;
		const v = this.map.get(key);

		if (v && v.expiry >= time) return v.value;
	}
	set(key: K, val: V): V {
		const time = Temporal.Now.instant().epochMilliseconds;
		this.map.set(key, { value: val, expiry: time + 30000 });
		return val;
	}
}

const author_cache = new cacher<`${string}/${string}`, message_author>();
const channel_cache = new cacher<string, Channel>();
const member_cache = new cacher<`${string}/${string}`, Member>();
const message_cache = new cacher<`${string}/${string}`, Message>();
const role_cache = new cacher<`${string}/${string}`, Role>();
const server_cache = new cacher<string, Server>();
const user_cache = new cacher<string, User>();

export async function fetch_author(
	api: Client,
	authorID: string,
	channelID: string,
	masquerade?: Masquerade,
): Promise<message_author> {
	try {
		const cached = author_cache.get(`${authorID}/${channelID}`);

		if (cached) return cached;

		const channel = await fetch_channel(api, channelID);
		const author = await fetch_user(api, authorID);

		const data = {
			id: authorID,
			rawname: author.username,
			username: masquerade?.name ?? author.username,
			color: masquerade?.colour ?? '#FF4654',
			profile: masquerade?.avatar ??
				(author.avatar
					? `https://autumn.revolt.chat/avatars/${author.avatar._id}`
					: undefined),
		};

		if (channel.channel_type !== 'TextChannel') return data;

		try {
			const member = await fetch_member(api, channel.server, authorID);

			return author_cache.set(`${authorID}/${channelID}`, {
				...data,
				username: masquerade?.name ?? member.nickname ?? data.username,
				profile: masquerade?.avatar ??
					(member.avatar
						? `https://autumn.revolt.chat/avatars/${member.avatar._id}`
						: data.profile),
			});
		} catch {
			return author_cache.set(`${authorID}/${channelID}`, data);
		}
	} catch {
		return {
			id: authorID,
			rawname: masquerade?.name ?? 'RevoltUser',
			username: masquerade?.name ?? 'Revolt User',
			profile: masquerade?.avatar ?? undefined,
			color: masquerade?.colour ?? '#FF4654',
		};
	}
}

export async function fetch_channel(
	api: Client,
	channelID: string,
): Promise<Channel> {
	const cached = channel_cache.get(channelID);

	if (cached) return cached;

	const channel = await api.request(
		'get',
		`/channels/${channelID}`,
		undefined,
	) as Channel;

	return channel_cache.set(channelID, channel);
}

export async function fetch_member(
	client: Client,
	serverID: string,
	userID: string,
): Promise<Member> {
	const member = member_cache.get(`${serverID}/${userID}`);

	if (member) return member;

	const response = await client.request(
		'get',
		`/servers/${serverID}/members/${userID}`,
		undefined,
	) as Member;

	return member_cache.set(`${serverID}/${userID}`, response);
}

export async function fetch_message(
	client: Client,
	channelID: string,
	messageID: string,
): Promise<Message> {
	const message = message_cache.get(`${channelID}/${messageID}`);

	if (message) return message;

	const response = await client.request(
		'get',
		`/channels/${channelID}/messages/${messageID}`,
		undefined,
	) as Message;

	return message_cache.set(`${channelID}/${messageID}`, response);
}

export async function fetch_role(
	client: Client,
	serverID: string,
	roleID: string,
): Promise<Role> {
	const role = role_cache.get(`${serverID}/${roleID}`);

	if (role) return role;

	const response = await client.request(
		'get',
		`/servers/${serverID}/roles/${roleID}`,
		undefined,
	) as Role;

	return role_cache.set(`${serverID}/${roleID}`, response);
}

export async function fetch_server(
	client: Client,
	serverID: string,
): Promise<Server> {
	const server = server_cache.get(serverID);

	if (server) return server;

	const response = await client.request(
		'get',
		`/servers/${serverID}`,
		undefined,
	) as Server;

	return server_cache.set(serverID, response);
}

export async function fetch_user(
	api: Client,
	userID: string,
): Promise<User> {
	const cached = user_cache.get(userID);

	if (cached) return cached;

	const user = await api.request(
		'get',
		`/users/${userID}`,
		undefined,
	) as User;

	return user_cache.set(userID, user);
}
