import { bridge_command } from './bridge/mod.ts';
import type { command } from './mod.ts';

export const default_commands = new Map([
    ['help', {
        name: 'help',
        description: 'get help with the bot',
        execute: () =>
            'check out [the docs](https://williamhorning.eu.org/bolt/) for help.',
    }],
    ['ping', {
        name: 'ping',
        description: 'check if the bot is alive',
        execute: ({ timestamp }) =>
            `Pong! 🏓 ${
                Temporal.Now.instant().since(timestamp).round('millisecond')
                    .total('milliseconds')
            }ms`,
    }],
    ['version', {
        name: 'version',
        description: 'get the bots version',
        execute: () => 'hello from v0.8.0!',
    }],
    ['bridge', bridge_command],
]) as Map<string, command>;
