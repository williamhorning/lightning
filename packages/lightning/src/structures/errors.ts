import { create_message, type message } from './messages.ts';

/** options used to create an error */
export interface error_options {
	/** the user-facing message of the error */
	message?: string;
	/** the extra data to log */
	extra?: Record<string, unknown>;
	/** whether to disable the associated channel (when bridging) */
	disable?: boolean;
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
	override cause: Error;
	/** extra information associated with the error */
	extra: Record<string, unknown>;
	/** the user-facing error message */
	msg: message;
	/** whether to disable the associated channel (when bridging) */
	disable_channel?: boolean;

	/** create and log an error */
	constructor(e: unknown, public options?: error_options) {
		if (e instanceof LightningError) {
			super(e.message, { cause: e.cause });
			this.id = e.id;
			this.cause = e.cause;
			this.extra = e.extra;
			this.msg = e.msg;
			this.disable_channel = e.disable_channel;
			return;
		}

		const cause = e instanceof Error
			? e
			: e instanceof Object
			? new Error(JSON.stringify(e))
			: new Error(String(e));

		const id = crypto.randomUUID();

		super(options?.message ?? cause.message, { cause });

		this.name = 'LightningError';
		this.id = id;
		this.cause = cause;
		this.extra = options?.extra ?? {};
		this.disable_channel = options?.disable;
		this.msg = create_message(
			`Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\`\`\`\n${this.message}\n${this.id}\n\`\`\``,
		);

		// the error-logging async fun
		(async () => {
			console.error(`%clightning error ${id}`, 'color: red');
			console.error(cause, this.options);

			const webhook = Deno.env.get('LIGHTNING_ERROR_WEBHOOK');

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
		})();
	}
}
