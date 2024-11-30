import { DiscordAPIError } from '@discordjs/rest';
import { log_error } from '@jersey/lightning';

export function error_handler(e: unknown, channel_id: string, action: string) {
	if (e instanceof DiscordAPIError) {
		if (e.code === 30007 || e.code === 30058) {
			log_error(e, {
				message: 'too many webhooks in channel/guild. try deleting some',
				extra: { channel_id },
			});
		} else if (e.code === 50013) {
			log_error(e, {
				message: 'missing permissions to create webhook. check bot permissions',
				extra: { channel_id },
			});
		} else if (e.code === 10003 || e.code === 10015 || e.code === 50027) {
			log_error(e, {
				disable: true,
				message: `disabling channel due to error code ${e.code}`,
				extra: { channel_id },
			});
		} else if (action === 'editing message' && e.code === 10008) {
			return []; // message already deleted or non-existent
		} else {
			log_error(e, {
				message: `unknown error ${action}`,
				extra: { channel_id, code: e.code },
			});
		}
	} else {
		log_error(e, {
			message: `unknown error ${action}`,
			extra: { channel_id },
		});
	}
}
