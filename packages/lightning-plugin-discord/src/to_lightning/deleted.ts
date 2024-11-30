import type { GatewayMessageDeleteDispatchData } from 'discord-api-types';
import type { deleted_message } from '@jersey/lightning';

export function deleted(
    message: GatewayMessageDeleteDispatchData,
): deleted_message {
    return {
        channel: message.channel_id,
        id: message.id,
        plugin: 'bolt-discord',
        timestamp: Temporal.Now.instant(),
    }
}
