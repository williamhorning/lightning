import { create_message, type message } from './messages.ts';

/** the error returned from log_error */
export interface err {
	/** id of the error */
	id: string;
	/** the original error */
	cause: Error;
	/** extra information about the error */
	extra: Record<string, unknown>;
	/** the message associated with the error */
	message: message;
}

/**
 * logs an error and returns a unique id and a message for users
 * @param e the error to log
 * @param extra any extra data to log
 */
export async function log_error(
	e: unknown,
	extra: Record<string, unknown> = {},
): Promise<err> {
	const id = crypto.randomUUID();
	const webhook = Deno.env.get('LIGHTNING_ERROR_HOOK');
	const cause = e instanceof Error
		? e
		: e instanceof Object
		? new Error(JSON.stringify(e))
		: new Error(String(e));
	const user_facing_text =
		`Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\`\`\`\n${cause.message}\n${id}\n\`\`\``;

	for (const key in extra) {
		if (key === 'lightning') {
			delete extra[key];
		}

		if (typeof extra[key] === 'object' && extra[key] !== null) {
			if ('lightning' in extra[key]) {
				delete extra[key].lightning;
			}
		}
	}

	// TODO(jersey): this is a really bad way of error handling-especially given it doesn't do a lot of stuff that would help debug errors-but it'll be replaced

	console.error(`%clightning error ${id}`, 'color: red');
	console.error(cause, extra);

	if (webhook && webhook.length > 0) {
		let json_str = `\`\`\`json\n${JSON.stringify(extra, null, 2)}\n\`\`\``;

		if (json_str.length > 2000) json_str = '*see console*';

		await fetch(webhook, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				content: `# ${cause.message}\n*${id}*`,
				embeds: [
					{
						title: 'extra',
						description: json_str,
					},
				],
			}),
		});
	}

	return { id, cause, extra, message: create_message(user_facing_text) };
}
