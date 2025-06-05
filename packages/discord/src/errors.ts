import { DiscordAPIError } from '@discordjs/rest';
import { log_error } from '@lightning/lightning';

const errors = [
	[30007, 'Too many webhooks in channel, try deleting some', false, true],
	[30058, 'Too many webhooks in guild, try deleting some', false, true],
	[50013, 'Missing permissions to make webhook', false, true],
	[10003, 'Unknown channel, disabling channel', true, true],
	[10015, 'Unknown message, disabling channel', false, true],
	[50027, 'Invalid webhook token, disabling channel', false, true],
	[0, 'Unknown DiscordAPIError, not disabling channel', false, false],
] as const;

export function handle_error(
	err: unknown,
	channel: string,
	edit?: boolean,
) {
	if (err instanceof DiscordAPIError) {
		if (edit && err.code === 10008) return []; // message already deleted or non-existent

		const extra = { channel, code: err.code };
		const [, message, read, write] = errors.find((e) => e[0] === err.code) ??
			errors[errors.length - 1];

		log_error(err, { disable: { read, write }, message, extra });
	} else {
		log_error(err, {
			message: `unknown discord error`,
			extra: { channel },
		});
	}
}
