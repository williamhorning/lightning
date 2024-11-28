import { create_message, type message } from './messages.ts';

export interface LightningErrorOptions {
	/** the user-facing message of the error */
	message?: string;
	/** the extra data to log */
	extra?: Record<string, unknown>;
}

export class LightningError extends Error {
	id: string;
	override cause: Error;
	extra: Record<string, unknown>;
	msg: message;

	constructor(e: unknown, options?: LightningErrorOptions) {
		if (e instanceof LightningError) {
			super(e.message, { cause: e.cause });
			this.id = e.id;
			this.cause = e.cause;
			this.extra = e.extra;
			this.msg = e.msg;
			return;
		}

		const cause = e instanceof Error
			? e
			: e instanceof Object
			? new Error(JSON.stringify(e))
			: new Error(String(e));
		
		super(options?.message ?? cause.message, { cause });

		this.name = 'LightningError';
		this.id = crypto.randomUUID();
		this.cause = cause;
		this.extra = options?.extra ?? {};
		this.msg = create_message(
			`Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\`\`\`\n${this.message}\n${this.id}\n\`\`\``
		);
		this.log();
	}

	log() {
		console.error(`%clightning error ${this.id}`, 'color: red');
		console.error(this.cause, this.extra);

		const webhook = Deno.env.get('LIGHTNING_ERROR_WEBHOOK');

		for (const key in this.extra) {
			if (key === 'lightning') {
				delete this.extra[key];
			}
	
			if (typeof this.extra[key] === 'object' && this.extra[key] !== null) {
				if ('lightning' in this.extra[key]) {
					delete this.extra[key].lightning;
				}
			}
		}

		if (webhook && webhook.length > 0) {
			let json_str = `\`\`\`json\n${JSON.stringify(this.extra, null, 2)}\n\`\`\``;

			if (json_str.length > 2000) json_str = '*see console*';

			fetch(webhook, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					content: `# ${this.cause.message}\n*${this.id}*`,
					embeds: [
						{
							title: 'extra',
							description: json_str,
						},
					],
				}),
			});
		}
	}
}

export function logError(e: unknown, options?: LightningErrorOptions): never {
	throw new LightningError(e, options);
}

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
	const error = new LightningError(e, { extra });

	return {
		id: error.id,
		cause: error.cause,
		extra: error.extra,
		message: error.msg,
	}
}
