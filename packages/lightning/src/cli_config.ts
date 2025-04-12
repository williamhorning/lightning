import {
	array,
	literal,
	number,
	object,
	optional,
	parse as parse_schema,
	record,
	string,
	union,
	unknown,
} from '@valibot/valibot';
import { parse as parse_toml } from '@std/toml';
import type { database_config } from './database/mod.ts';
import type { core_config } from './core.ts';

const cli_config = object({
	database: union([
		object({
			type: literal('postgres'),
			config: string(),
		}),
		object({
			type: literal('redis'),
			config: object({
				port: number(),
				hostname: optional(string()),
			}),
		}),
	]),
	error_url: optional(string()),
	prefix: optional(string(), '!'),
	plugins: array(object({
		plugin: string(),
		config: record(string(), unknown()),
	})),
});

export interface config extends core_config {
	database: database_config;
	error_url?: string;
}

// TODO: error handle
export async function parse_config(
	path: string,
): Promise<config> {
	const file = await Deno.readTextFile(path);
	const raw = parse_toml(file);
	const parsed = parse_schema(cli_config, raw);
	const new_plugins = [];

	for (const plugin of parsed.plugins) {
		new_plugins.push({
			module: await import(plugin.plugin),
			config: plugin.config,
		});
	}

	Deno.env.set('LIGHTNING_ERROR_WEBHOOK', parsed.error_url || '');

	return { ...parsed, plugins: new_plugins };
}
