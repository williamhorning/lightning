import { DiscordAPIError } from '@discordjs/rest';
import { log_error } from '@jersey/lightning';

export function handle_error(
	err: unknown,
	channel: string,
	edit?: boolean,
) {
	if (err instanceof DiscordAPIError) {
		if (err.code === 30007 || err.code === 30058) {
			log_error(err, {
				message: 'too many webhooks in channel/guild. try deleting some',
				extra: { channel },
			});
		} else if (err.code === 50013) {
			log_error(err, {
				message: 'missing permissions to create webhook. check bot permissions',
				extra: { channel },
			});
		} else if (err.code === 10003 || err.code === 10015 || err.code === 50027) {
			log_error(err, {
				disable: true,
				message: `disabling channel due to error code ${err.code}`,
				extra: { channel },
			});
		} else if (edit && err.code === 10008) {
			return []; // message already deleted or non-existent
		} else {
			log_error(err, {
				message: `unknown discord api error`,
				extra: { channel, code: err.code },
			});
		}
	} else {
		log_error(err, {
			message: `unknown discord error`,
			extra: { channel },
		});
	}
}
