import { RequestError } from '@jersey/revolt-api-types';
import { MediaError } from '@jersey/rvapi';
import { log_error } from '@lightning/lightning';

const errors = [
	[403, 'Insufficient permissions. Please check them', true],
	[404, 'Resource not found', true],
	[0, 'Unknown Revolt RequestError', false],
] as const;

export function handle_error(err: unknown, edit?: boolean): never[] {
	if (err instanceof MediaError) {
		log_error(err);
	} else if (err instanceof RequestError) {
		if (err.cause.status === 404 && edit) return [];

		const [, message, disable] = errors.find((e) =>
			e[0] === err.cause.status
		) ?? errors[errors.length - 1];

		log_error(err, { message, disable });
	} else {
		log_error(err, { message: 'unknown revolt error' });
	}
}
