import { Client } from '@discordjs/core';
import { REST } from '@discordjs/rest';
import { WebSocketManager } from '@discordjs/ws';
import {
    type create_message_opts,
    type delete_message_opts,
    type edit_message_opts,
    type lightning,
    plugin,
} from '@jersey/lightning';
import { GatewayDispatchEvents } from 'discord-api-types';
import * as bridge from './bridge_to_discord.ts';
import { setup_slash_commands } from './slash_commands.ts';
import { command_to } from './to_lightning/command.ts';
import { deleted } from './to_lightning/deleted.ts';
import { message } from './to_lightning/message.ts';

/** configuration for the discord plugin */
export interface discord_config {
    /** the discord bot token */
    token: string;
    /** whether to enable slash commands */
    slash_commands: boolean;
    /** discord application id */
    application_id: string;
}

export class discord_plugin extends plugin<discord_config> {
    name = 'bolt-discord';
    private api: Client['api'];
    private client: Client;

    constructor(l: lightning, config: discord_config) {
        super(l, config);
        // @ts-ignore their type for makeRequest is funky
        const rest = new REST({ version: '10', makeRequest: fetch }).setToken(
            config.token,
        );
        const gateway = new WebSocketManager({
            token: config.token,
            intents: 0 | 33281,
            rest,
        });
        // @ts-ignore Deno is wrong here.
        this.client = new Client({ rest, gateway });
        this.api = this.client.api;

        setup_slash_commands(this.api, config, l);
        this.setup_events();
        gateway.connect();
    }

    private setup_events() {
        // @ts-ignore I'm going to file an issue against Deno because this is so annoying
        this.client.once(GatewayDispatchEvents.Ready, (ev) => {
            console.log(
                `bolt-discord: ready as ${ev.user.username}#${ev.user.discriminator} in ${ev.guilds.length} guilds`,
            );
        });
        // @ts-ignore see above
        this.client.on(GatewayDispatchEvents.MessageCreate, async (msg) => {
            this.emit('create_message', await message(msg.api, msg.data));
        });
        // @ts-ignore see above
        this.client.on(GatewayDispatchEvents.MessageUpdate, async (msg) => {
            this.emit('edit_message', await message(msg.api, msg.data));
        });
        // @ts-ignore see above
        this.client.on(GatewayDispatchEvents.MessageDelete, (msg) => {
            this.emit('delete_message', deleted(msg.data));
        });
        // @ts-ignore see above
        this.client.on(GatewayDispatchEvents.InteractionCreate, (cmd) => {
            const command = command_to(cmd, this.lightning);
            if (command) this.emit('create_command', command);
        });
    }

    async setup_channel(channel: string) {
        return await bridge.setup_bridge(this.api, channel);
    }

    async create_message(opts: create_message_opts) {
        return await bridge.create_message(this.api, opts);
    }

    async edit_message(opts: edit_message_opts) {
        return await bridge.edit_message(this.api, opts);
    }

    async delete_message(opts: delete_message_opts) {
        return await bridge.delete_message(this.api, opts);
    }
}
