import { log_error } from '@jersey/lightning';
import { GuildedAPIError } from 'guilded.js';

export function error_handler(e: unknown, channel_id: string, action: string) {
	if (e instanceof GuildedAPIError) {
		if (e.response.status === 404) {
            if (action === 'deleting message') return [];

            log_error(e, {
                message: 'resource not found! if you\'re trying to make a bridge, this is likely an issue with Guilded',
                extra: { channel_id, response: e.response },
                disable: true,
            });
        } else if (e.response.status === 403) {
            log_error(e, {
                message: 'no permission to send/delete messages! check bot permissions',
                extra: { channel_id, response: e.response },
                disable: true,
            });
        } else {
            log_error(e, {
                message: `unknown guilded error ${action} with status code ${e.response.status}`,
                extra: { channel_id, response: e.response }
            })
        }
	} else {
		log_error(e, {
			message: `unknown error ${action}`,
			extra: { channel_id },
		});
	}
}
