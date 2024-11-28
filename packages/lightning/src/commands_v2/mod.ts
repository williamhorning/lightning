import { bridge_command } from './bridge/mod.ts';
import type { lightning } from '../lightning.ts';
import { create_message, type message } from '../messages.ts';

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
    subcommands?: command[];
    // TODO(jersey): message | string | Promise<message> | Promise<string>
    execute: (opts: command_execute_options) => Promise<message> | message;
}

export const default_commands = [
    ['help', {
        name: 'help',
        description: 'get help with the bot',
        execute: () =>
            create_message(
                'check out [the docs](https://williamhorning.eu.org/bolt/) for help.',
            ),
    }],
    ['ping', {
        name: 'ping',
        description: 'check if the bot is alive',
        execute: ({ timestamp }) => {
            const diff = Temporal.Now.instant().since(timestamp).total(
                'milliseconds',
            );
            return create_message(`Pong! 🏓 ${diff}ms`);
        },
    }],
    ['version', {
        name: 'version',
        description: 'get the bots version',
        execute: () => create_message('hello from v0.8.0!'),
    }],
    ['bridge', bridge_command],
] as [string, command][];

// TODO(jersey): make command runners
