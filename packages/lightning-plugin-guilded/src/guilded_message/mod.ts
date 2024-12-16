import type { message } from '@jersey/lightning';
import type { Client, Message } from 'guilded.js';
import { convert_msg } from '../guilded.ts';
import { get_author } from './author.ts';
import { map_embed } from './map_embed.ts';

export async function guilded_to_message(
    msg: Message,
    bot: Client,
): Promise<message | undefined> {
    if (msg.serverId === null) return;

    const author = await get_author(msg, bot);

    const timestamp = Temporal.Instant.fromEpochMilliseconds(
        msg.createdAt.valueOf(),
    );

    return {
            author: {
                ...author,
                color: '#F5C400',
            },
            channel: msg.channelId,
            id: msg.id,
            timestamp,
            embeds: msg.embeds?.map(map_embed),
            plugin: 'bolt-guilded',
            reply: async (reply: message) => {
                await msg.reply(await convert_msg(reply));
            },
            content: msg.content.replaceAll('\n```\n```\n', '\n'),
            reply_id: msg.isReply ? msg.replyMessageIds[0] : undefined,
        };
    
}
