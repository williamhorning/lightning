import { getEnv } from '@cross/env';
import { create_message, type message } from './messages.ts';

/** options used to create an error */
export interface error_options {
	/** the user-facing message of the error */
	message?: string;
	/** the extra data to log */
	extra?: Record<string, unknown>;
	/** whether to disable the associated channel (when bridging) */
	disable?: boolean;
	/** whether this should be logged without the cause */
	without_cause?: boolean;
}

/** logs an error */
export function log_error(e: unknown, options?: error_options): never {
	throw new LightningError(e, options);
}

/** lightning error */
export class LightningError extends Error {
	/** the id associated with the error */
	id: string;
	/** the cause of the error */
	private error_cause: Error;
	/** extra information associated with the error */
	extra: Record<string, unknown>;
	/** the user-facing error message */
	msg: message;
	/** whether to disable the associated channel (when bridging) */
	disable_channel?: boolean;
	/** whether to show the cause or not */
	without_cause?: boolean;

	/** create and log an error */
	constructor(e: unknown, public options?: error_options) {
		if (e instanceof LightningError) {
			super(e.message, { cause: e.cause });
			this.id = e.id;
			this.error_cause = e.error_cause;
			this.extra = e.extra;
			this.msg = e.msg;
			this.disable_channel = e.disable_channel;
			this.without_cause = e.without_cause;
			return e;
		}

		const cause_err = Error.isError(e)
			? e
			: e instanceof Object
			? new Error(JSON.stringify(e))
			: new Error(String(e));

		const id = crypto.randomUUID();

		super(options?.message ?? cause_err.message, { cause: e });

		this.name = 'LightningError';
		this.id = id;
		this.error_cause = cause_err;
		this.extra = options?.extra ?? {};
		this.disable_channel = options?.disable;
		this.without_cause = options?.without_cause;
		this.msg = create_message(
			`Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\`\`\`\n${this.message}\n${this.id}\n\`\`\``,
		);

		this.log();
	}

	/** log the error */
	private async log(): Promise<void> {
		console.error(`%c[lightning] ${this.message}`, 'color: red');
		console.error(`%c[lightning] ${this.id}`, 'color: red');
		console.error(
			`%c[lightning] this does${
				this.disable_channel ? ' ' : ' not '
			}disable a channel`,
			'color: red',
		);

		if (!this.without_cause) console.error(this.error_cause, this.extra);

		const webhook = getEnv('LIGHTNING_ERROR_WEBHOOK');

		for (const key in this.options?.extra) {
			if (key === 'lightning') {
				delete this.options.extra[key];
			}

			if (
				typeof this.options.extra[key] === 'object' &&
				this.options.extra[key] !== null
			) {
				if ('lightning' in this.options.extra[key]) {
					delete this.options.extra[key].lightning;
				}
			}
		}

		if (webhook && webhook.length > 0) {
			let json_str = `\`\`\`json\n${
				JSON.stringify(this.options?.extra, null, 2)
			}\n\`\`\``;

			if (json_str.length > 2000) json_str = '*see console*';

			await fetch(webhook, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					content: `# ${this.error_cause.message}\n*${this.id}*`,
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
