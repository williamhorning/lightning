import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	plugin,
} from '@jersey/lightning';
import { type Client, createClient } from '@jersey/rvapi';
import type { Message } from '@jersey/revolt-api-types';
import { handle_error } from './errors.ts';
import { check_permissions } from './permissions.ts';
import { get_revolt_message } from './messages.ts';
import { setup_events } from './events.ts';

/** the config for the revolt plugin */
export interface revolt_config {
	/** the token for the revolt bot */
	token: string;
	/** the user id for the bot */
	user_id: string;
}

/** the plugin to use */
export class revolt_plugin extends plugin<revolt_config> {
	bot: Client;
	name = 'bolt-revolt';

	constructor(l: lightning, config: revolt_config) {
		super(l, config);
		this.bot = createClient(config);
		setup_events(this.bot, config, this.emit);
	}

	async setup_channel(channel: string): Promise<unknown> {
		return await check_permissions(channel, this.bot, this.config.user_id);
	}

	async create_message(opts: create_opts): Promise<string[]> {
		try {
			const { _id } = (await this.bot.request(
				'post',
				`/channels/${opts.channel.id}/messages`,
				await get_revolt_message(this.bot, opts.msg, true),
			)) as Message;

			return [_id];
		} catch (e) {
			return handle_error(e);
		}
	}

	async edit_message(opts: edit_opts): Promise<string[]> {
		try {
			await this.bot.request(
				'patch',
				`/channels/${opts.channel.id}/messages/${opts.edit_ids[0]}`,
				await get_revolt_message(this.bot, opts.msg, true),
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, true);
		}
	}

	async delete_message(opts: delete_opts): Promise<string[]> {
		try {
			await this.bot.request(
				'delete',
				`/channels/${opts.channel.id}/messages/${opts.edit_ids[0]}`,
				undefined,
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, true);
		}
	}
}
