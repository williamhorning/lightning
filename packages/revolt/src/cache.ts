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
import type { message_author } from '@jersey/lightning';

class RevoltCacher<K extends string, V> {
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

const authorCache = new RevoltCacher<`${string}/${string}`, message_author>();
const channelCache = new RevoltCacher<string, Channel>();
const memberCache = new RevoltCacher<`${string}/${string}`, Member>();
const messageCache = new RevoltCacher<`${string}/${string}`, Message>();
const roleCache = new RevoltCacher<`${string}/${string}`, Role>();
const serverCache = new RevoltCacher<string, Server>();
const userCache = new RevoltCacher<string, User>();

export async function fetchAuthor(
	api: Client,
	authorID: string,
	channelID: string,
	masquerade?: Masquerade,
): Promise<message_author> {
	try {
		const cached = authorCache.get(`${authorID}/${channelID}`);

		if (cached) return cached;

		const channel = await fetchChannel(api, channelID);
		const author = await fetchUser(api, authorID);

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
			const member = await fetchMember(api, channel.server, authorID);

			return authorCache.set(`${authorID}/${channelID}`, {
				...data,
				username: masquerade?.name ?? member.nickname ?? data.username,
				profile: masquerade?.avatar ??
					(member.avatar
						? `https://autumn.revolt.chat/avatars/${member.avatar._id}`
						: data.profile),
			});
		} catch {
			return authorCache.set(`${authorID}/${channelID}`, data);
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

export async function fetchChannel(
	api: Client,
	channelID: string,
): Promise<Channel> {
	const cached = channelCache.get(channelID);

	if (cached) return cached;

	const channel = await api.request(
		'get',
		`/channels/${channelID}`,
		undefined,
	) as Channel;

	return channelCache.set(channelID, channel);
}

export async function fetchMember(
	client: Client,
	serverID: string,
	userID: string,
): Promise<Member> {
	const member = memberCache.get(`${serverID}/${userID}`);

	if (member) return member;

	const response = await client.request(
		'get',
		`/servers/${serverID}/members/${userID}`,
		undefined,
	) as Member;

	return memberCache.set(`${serverID}/${userID}`, response);
}

export async function fetchMessage(
	client: Client,
	channelID: string,
	messageID: string,
): Promise<Message> {
	const message = messageCache.get(`${channelID}/${messageID}`);

	if (message) return message;

	const response = await client.request(
		'get',
		`/channels/${channelID}/messages/${messageID}`,
		undefined,
	) as Message;

	return messageCache.set(`${channelID}/${messageID}`, response);
}

export async function fetchRole(
	client: Client,
	serverID: string,
	roleID: string,
): Promise<Role> {
	const role = roleCache.get(`${serverID}/${roleID}`);

	if (role) return role;

	const response = await client.request(
		'get',
		`/servers/${serverID}/roles/${roleID}`,
		undefined,
	) as Role;

	return roleCache.set(`${serverID}/${roleID}`, response);
}

export async function fetchServer(
	client: Client,
	serverID: string,
): Promise<Server> {
	const server = serverCache.get(serverID);

	if (server) return server;

	const response = await client.request(
		'get',
		`/servers/${serverID}`,
		undefined,
	) as Server;

	return serverCache.set(serverID, response);
}

export async function fetchUser(
	api: Client,
	userID: string,
): Promise<User> {
	const cached = userCache.get(userID);

	if (cached) return cached;

	const user = await api.request(
		'get',
		`/users/${userID}`,
		undefined,
	) as User;

	return userCache.set(userID, user);
}
