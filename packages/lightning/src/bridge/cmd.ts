import type { command } from '../commands/mod.ts';
import { log_error } from '../errors.ts';

export const bridge_command = {
	name: 'bridge',
	description: 'bridge commands',
	execute: () => 'take a look at the docs for help with bridges',
	options: {
		subcommands: [
			// TODO(jersey): eventually reimplement reset command?
			{
				name: 'join',
				description: 'join a bridge',
				options: { argument_name: 'name', argument_required: true },
				execute: async ({ lightning, channel, opts, plugin }) => {
					const current_bridge = await lightning.data
						.get_bridge_by_channel(
							channel,
						);

					// live laugh love validation

					if (current_bridge) {
						return `You are already in a bridge called ${current_bridge.name}`;
					}
					if (opts.id && opts.name) {
						return `You can only provide an id or a name, not both`;
					}
					if (!opts.id && !opts.name) {
						return `You must provide either an id or a name`;
					}

					const bridge_channel = {
						id: channel,
						data: undefined as unknown,
						disabled: false,
						plugin,
					};

					try {
						bridge_channel.data = lightning.plugins.get(plugin)
							?.create_bridge(channel);
					} catch (e) {
						return (await log_error(
							new Error('error creating bridge', { cause: e }),
							{
								channel,
								plugin_name: plugin,
							},
						)).message.content as string;
					}

					if (opts.id) {
						const bridge = await lightning.data.get_bridge_by_id(
							opts.id,
						);

						if (!bridge) return `No bridge found with that id`;

						bridge.channels.push(bridge_channel);

						try {
							await lightning.data.update_bridge(bridge);
							return `Bridge joined successfully`;
						} catch (e) {
							return (await log_error(
								new Error('error updating bridge', { cause: e }),
								{
									bridge,
								},
							)).message.content as string;
						}
					} else {
						try {
							await lightning.data.new_bridge({
								name: opts.name,
								channels: [bridge_channel],
								settings: {
									allow_editing: true,
									allow_everyone: false,
									use_rawname: false,
								},
							});
							return `Bridge joined successfully`;
						} catch (e) {
							return (await log_error(
								new Error('error inserting bridge', { cause: e }),
								{
									bridge: {
										name: opts.name,
										channels: [bridge_channel],
										settings: {
											allow_editing: true,
											allow_everyone: false,
											use_rawname: false,
										},
									},
								},
							)).message.content as string;
						}
					}
				},
			},
			{
				name: 'leave',
				description: 'leave a bridge',
				execute: async ({ lightning, channel }) => {
					const bridge = await lightning.data.get_bridge_by_channel(
						channel,
					);

					if (!bridge) return `You are not in a bridge`;

					bridge.channels = bridge.channels.filter((
						ch,
					) => ch.id !== channel);

					try {
						await lightning.data.update_bridge(
							bridge,
						);
						return `Bridge left successfully`;
					} catch (e) {
						return await log_error(
							new Error('error updating bridge', { cause: e }),
							{
								bridge,
							},
						);
					}
				},
			},
			{
				name: 'toggle',
				description: 'toggle a setting on a bridge',
				options: { argument_name: 'setting', argument_required: true },
				execute: async ({ opts, lightning, channel }) => {
					const bridge = await lightning.data.get_bridge_by_channel(
						channel,
					);

					if (!bridge) return `You are not in a bridge`;

					if (
						!['allow_editing', 'allow_everyone', 'use_rawname']
							.includes(opts.setting)
					) {
						return `that setting does not exist`;
					}

					const key = opts.setting as keyof typeof bridge.settings;

					bridge.settings[key] = !bridge
						.settings[key];

					try {
						await lightning.data.update_bridge(
							bridge,
						);
						return `Setting toggled successfully`;
					} catch (e) {
						return await log_error(
							new Error('error updating bridge', { cause: e }),
							{
								bridge,
							},
						);
					}
				},
			},
			{
				name: 'status',
				description: 'see what bridges you are in',
				execute: async ({ lightning, channel }) => {
					const existing_bridge = await lightning.data
						.get_bridge_by_channel(
							channel,
						);

					if (!existing_bridge) return `You are not in a bridge`;

					return `You are in a bridge called ${existing_bridge.name} that's connected to ${
						existing_bridge.channels.length - 1
					} other channels`;
				},
			},
		],
	},
} as command;
