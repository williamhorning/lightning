import type { command_arguments } from './mod.ts';
import { create_message, type message } from '../messages.ts';
import { log_error } from '../errors.ts';
import type { lightning } from '../lightning.ts';
import { parseArgs } from '@std/cli/parse-args';

export function handle_command_message(m: message, l: lightning) {
	if (!m.content?.startsWith(l.config.cmd_prefix)) return;

	const {
		_: [cmd, subcmd],
		...opts
	} = parseArgs(m.content.replace(l.config.cmd_prefix, '').split(' '));

	run_command({
		lightning: l,
		cmd: cmd as string,
		subcmd: subcmd as string,
		opts,
		...m,
	});
}

export async function run_command(args: command_arguments) {
	let reply;

	try {
		const cmd = args.lightning.commands.get(args.cmd) ||
			args.lightning.commands.get('help')!;

		const exec = cmd.options?.subcommands?.find((i) =>
			i.name === args.subcmd
		)?.execute ||
			cmd.execute;

		reply = create_message(await exec(args));
	} catch (e) {
		reply = (await log_error(e, { ...args, reply: undefined })).message;
	}

	try {
		await args.reply(reply, false);
	} catch (e) {
		await log_error(e, { ...args, reply: undefined });
	}
}
