import { RequestError } from '@jersey/guilded-api-types';
import { log_error } from '@lightning/lightning';

export function handle_error(err: unknown, channel: string): never {
	if (err instanceof RequestError) {
		if (err.cause.status === 404) {
			log_error(err, {
				message:
					"resource not found! if you're trying to make a bridge, this is likely an issue with Guilded",
				extra: { channel_id: channel, response: err.cause },
				disable: true,
			});
		} else if (err.cause.status === 403) {
			log_error(err, {
				message: 'no permission to send/delete messages! check bot permissions',
				extra: { channel_id: channel, response: err.cause },
				disable: true,
			});
		} else {
			log_error(err, {
				message: `unknown guilded error with status code ${err.cause.status}`,
				extra: { channel_id: channel, response: err.cause },
			});
		}
	} else {
		log_error(err, {
			message: `unknown error`,
			extra: { channel_id: channel },
		});
	}
}
