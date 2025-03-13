import { MediaError, RequestError } from '@jersey/rvapi';
import { log_error } from '@jersey/lightning';

export function handle_error(err: unknown, edit?: boolean) {
	if (err instanceof MediaError) {
		log_error(err, {
			message: err.message,
		});
	} else if (err instanceof RequestError) {
		if (err.cause.status === 403) {
			log_error(err, {
				message: 'Insufficient permissions',
				disable: true,
			});
		} else if (err.cause.status === 404) {
			if (edit) return [];

			log_error(err, {
				message: 'Resource not found',
				disable: true,
			});
		} else {
			log_error(err, {
				message: 'unknown error',
			});
		}
	} else {
		log_error(err, {
			message: 'unknown error',
		});
	}
}
