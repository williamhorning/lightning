import type { APIMessage } from 'discord-api-types';
import type { API } from '@discordjs/core';
import { calculateUserDefaultAvatarIndex } from '@discordjs/rest';

export async function get_author(api: API, message: APIMessage) {
    let name = message.author.global_name || message.author.username;
    let avatar = message.author.avatar ? `https://cdn.discordapp.com/avatars/${message.author.id}/${message.author.avatar}.png` : `https://cdn.discordapp.com/embed/avatars/${calculateUserDefaultAvatarIndex(message.author.id)}.png`;

    const channel = await api.channels.get(message.channel_id);
    
    if ("guild_id" in channel) {
        try {
            const member = await api.guilds.getMember(channel.guild_id, message.author.id);

            name ??= member.nick;
            avatar = member.avatar ? `https://cdn.discordapp.com/guilds/${channel.guild_id}/users/${message.author.id}/avatars/${member.avatar}.png` : avatar;
        } catch {
            // safe to ignore
        }
    }

    return { name, avatar }
}
