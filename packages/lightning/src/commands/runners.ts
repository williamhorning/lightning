import type { lightning } from '../lightning.ts';
import {
	type create_command,
	create_message,
	LightningError,
	type message,
} from '../structures/mod.ts';

export async function execute_text_command(msg: message, lightning: lightning) {
	if (!msg.content?.startsWith(lightning.config.prefix)) return;

	const [cmd, ...rest] = msg.content.replace(lightning.config.prefix, '')
		.split(' ');

	return await run_command({
		...msg,
		lightning,
		command: cmd as string,
		rest: rest as string[],
	});
}

export async function run_command(
	opts: create_command,
) {
	let command = opts.lightning.commands.get(opts.command) ??
		opts.lightning.commands.get('help')!;

	const subcommand_name = opts.subcommand ?? opts.rest?.shift();

	if (command.subcommands && subcommand_name) {
		const subcommand = command.subcommands.find((i) =>
			i.name === subcommand_name
		);

		if (subcommand) command = subcommand;
	}

	if (!opts.args) opts.args = {};

	for (const arg of command.arguments || []) {
		if (!opts.args[arg.name]) {
			opts.args[arg.name] = opts.rest?.shift() as string;
		}

		if (!opts.args[arg.name]) {
			return opts.reply(
				create_message(
					`Please provide the \`${arg.name}\` argument. Try using the \`${opts.lightning.config.prefix}help\` command.`,
				),
				false,
			);
		}
	}

	let resp: string | LightningError;

	try {
		resp = await command.execute({
			...opts,
			args: opts.args,
		});
	} catch (e) {
		if (e instanceof LightningError) resp = e;
		else {resp = new LightningError(e, {
				message: 'An error occurred while executing the command',
				extra: { command: command.name },
			});}
	}

	try {
		if (typeof resp === 'string') {
			await opts.reply(create_message(resp), false);
		} else await opts.reply(resp.msg, false);
	} catch (e) {
		new LightningError(e, {
			message: 'An error occurred while sending the command response',
			extra: { command: command.name },
		});
	}
}
