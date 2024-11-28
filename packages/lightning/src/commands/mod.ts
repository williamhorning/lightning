import type { lightning } from '../lightning.ts';
import { create_message, type message } from '../messages.ts';
import { LightningError } from '../errors.ts';

export interface command_execute_options {
    channel: string;
    plugin: string;
    timestamp: Temporal.Instant;
    arguments: Record<string, string>;
    lightning: lightning;
    id: string;
}

export interface command {
    name: string;
    description: string;
    arguments?: {
        name: string;
        description: string;
        required: boolean;
    }[];
    subcommands?: Omit<command, 'subcommands'>[];
    execute: (
        opts: command_execute_options,
    ) => Promise<string> | string;
}

export async function execute_text_command(msg: message, lightning: lightning) {
    if (!msg.content?.startsWith(lightning.config.cmd_prefix)) return;

    const [cmd, ...rest] = msg.content.replace(lightning.config.cmd_prefix, '').split(' ');

    return await run_command({
        ...msg,
        lightning,
        command: cmd as string,
        rest: rest as string[],
    });
}

export interface run_command_options
    extends Omit<command_execute_options, 'arguments'> {
    command: string;
    subcommand?: string;
    args?: Record<string, string>;
    rest?: string[];
    reply: message['reply'];
}

export async function run_command(
    opts: run_command_options,
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
                    `Please provide the \`${arg.name}\` argument. Try using the \`${opts.lightning.config.cmd_prefix}help\` command.`,
                ),
                false,
            );
        }
    }

    let resp: string | LightningError;

    try {
        resp = await command.execute({
            ...opts,
            arguments: opts.args,
        });
    } catch (e) {
        if (e instanceof LightningError) resp = e;
        else resp = new LightningError(e, {
            message: 'An error occurred while executing the command',
            extra: { command: command.name },
        })
    }

    try {
        if (typeof resp === 'string') {
            await opts.reply(create_message(resp), false);
        } else await opts.reply(resp.msg, false);
    } catch (e) {
        new LightningError(e, {
            message: 'An error occurred while sending the command response',
            extra: { command: command.name },
        })
    }
}
