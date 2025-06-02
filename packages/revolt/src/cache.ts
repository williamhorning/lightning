import type {
	Channel,
	Emoji,
	Masquerade,
	Member,
	Message,
	Role,
	Server,
	User,
} from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import { cacher, type message_author } from '@lightning/lightning';

const authors = new cacher<`${string}/${string}`, message_author>();
const channels = new cacher<string, Channel>();
const emojis = new cacher<string, Emoji>();
const members = new cacher<`${string}/${string}`, Member>();
const messages = new cacher<`${string}/${string}`, Message>();
const roles = new cacher<`${string}/${string}`, Role>();
const servers = new cacher<string, Server>();
const users = new cacher<string, User>();

export async function fetch_author(
	api: Client,
	authorID: string,
	channelID: string,
	masquerade?: Masquerade,
): Promise<message_author> {
	try {
		const cached = authors.get(`${authorID}/${channelID}`);

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
					? `https://cdn.revoltusercontent.com/avatars/${author.avatar._id}`
					: undefined),
		};

		if (channel.channel_type !== 'TextChannel') return data;

		try {
			const member = await fetch_member(api, channel.server, authorID);

			return authors.set(`${authorID}/${channelID}`, {
				...data,
				username: masquerade?.name ?? member.nickname ?? data.username,
				profile: masquerade?.avatar ??
					(member.avatar
						? `https://cdn.revoltusercontent.com/avatars/${member.avatar._id}`
						: data.profile),
			});
		} catch {
			return authors.set(`${authorID}/${channelID}`, data);
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
	const cached = channels.get(channelID);

	if (cached) return cached;

	const channel = await api.request(
		'get',
		`/channels/${channelID}`,
		undefined,
	) as Channel;

	return channels.set(channelID, channel);
}

export async function fetch_emoji(
	api: Client,
	emoji_id: string,
): Promise<Emoji> {
	const cached = emojis.get(emoji_id);

	if (cached) return cached;

	return emojis.set(
		emoji_id,
		await api.request(
			'get',
			`/custom/emoji/${emoji_id}`,
			undefined,
		),
	);
}

export async function fetch_member(
	client: Client,
	serverID: string,
	userID: string,
): Promise<Member> {
	const member = members.get(`${serverID}/${userID}`);

	if (member) return member;

	const response = await client.request(
		'get',
		`/servers/${serverID}/members/${userID}`,
		{ roles: false },
	) as Member;

	return members.set(`${serverID}/${userID}`, response);
}

export async function fetch_message(
	client: Client,
	channelID: string,
	messageID: string,
): Promise<Message> {
	const message = messages.get(`${channelID}/${messageID}`);

	if (message) return message;

	const response = await client.request(
		'get',
		`/channels/${channelID}/messages/${messageID}`,
		undefined,
	) as Message;

	return messages.set(`${channelID}/${messageID}`, response);
}

export async function fetch_role(
	client: Client,
	serverID: string,
	roleID: string,
): Promise<Role> {
	const role = roles.get(`${serverID}/${roleID}`);

	if (role) return role;

	const response = await client.request(
		'get',
		`/servers/${serverID}/roles/${roleID}`,
		undefined,
	) as Role;

	return roles.set(`${serverID}/${roleID}`, response);
}

export async function fetch_server(
	client: Client,
	serverID: string,
): Promise<Server> {
	const server = servers.get(serverID);

	if (server) return server;

	const response = await client.request(
		'get',
		`/servers/${serverID}`,
		undefined,
	) as Server;

	return servers.set(serverID, response);
}

export async function fetch_user(
	api: Client,
	userID: string,
): Promise<User> {
	const cached = users.get(userID);

	if (cached) return cached;

	const user = await api.request(
		'get',
		`/users/${userID}`,
		undefined,
	) as User;

	return users.set(userID, user);
}
