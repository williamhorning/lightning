import { RequestError } from '@jersey/guilded-api-types';
import { log_error } from '@lightning/lightning';

const errors = [
	[403, 'No permission to send/delete messages! Check permissions', true],
	[404, 'Not found! This might be a Guilded problem if making a bridge', true],
	[0, 'Unknown Guilded error, not disabling channel', false],
] as const;

export function handle_error(err: unknown, channel: string): never {
	if (err instanceof RequestError) {
		const [, message, disable] = errors.find((e) =>
			e[0] === err.cause.status
		) ??
			errors[errors.length - 1];

		log_error(err, {
			disable,
			extra: { channel_id: channel, response: err.cause },
			message,
		});
	} else {
		log_error(err, {
			message: `unknown error`,
			extra: { channel_id: channel },
		});
	}
}
