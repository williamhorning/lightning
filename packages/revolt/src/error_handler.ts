import { MediaError, RequestError } from '@jersey/rvapi';
import { log_error } from '@jersey/lightning';

export function handle_error(e: unknown, edit?: boolean) {
	if (e instanceof MediaError) {
		log_error(e, {
			message: e.message,
		});
	} else if (e instanceof RequestError) {
		if (e.cause.status === 403) {
			log_error(e, {
				message: 'Insufficient permissions',
				disable: true,
			});
		} else if (e.cause.status === 404) {
			if (edit) return [];

			log_error(e, {
				message: 'Resource not found',
				disable: true,
			});
		} else {
			log_error(e, {
				message: 'unknown error',
			});
		}
	} else {
		log_error(e, {
			message: 'unknown error',
		});
	}
}
