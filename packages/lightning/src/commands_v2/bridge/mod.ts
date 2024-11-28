import { create, join } from './add.ts';
import { leave, toggle } from './modify.ts';
import { status } from './status.ts';
import type { command } from '../mod.ts';
import { create_message } from '../../messages.ts';

export const bridge_command = {
    name: 'bridge',
    description: 'bridge commands',
    execute: () =>
        create_message('take a look at the subcommands of this command'),
    subcommands: [
        {
            name: 'create',
            description: 'create a new bridge',
            arguments: [{
                name: 'name',
                description: 'name of the bridge',
                required: true,
            }],
            execute: create,
        },
        {
            name: 'join',
            description: 'join an existing bridge',
            arguments: [{
                name: 'id',
                description: 'id of the bridge',
                required: true,
            }],
            execute: join,
        },
        {
            name: 'leave',
            description: 'leave the current bridge',
            execute: leave,
        },
        {
            name: 'toggle',
            description: 'toggle a setting on the current bridge',
            arguments: [{
                name: 'setting',
                description: 'setting to toggle',
                required: true,
            }],
            execute: toggle,
        },
        {
            name: 'status',
            description: 'get the status of the current bridge',
            execute: status,
        },
    ],
} as command;
