import { readTextFile } from '@std/fs/unstable-read-text-file';
import { parse as parse_toml } from '@std/toml';
import type { core_config } from './core.ts';
import type { database_config } from './database/mod.ts';
import { set_env } from './structures/cross.ts';
import { log_error } from './structures/errors.ts';
import { validate_config } from './structures/validate.ts';

interface cli_plugin {
	plugin: string;
	config: Record<string, unknown>;
}

interface config extends core_config {
	database: database_config;
	error_url?: string;
}

export async function parse_config(path: URL): Promise<config> {
	try {
		const file = await readTextFile(path);
		const raw = parse_toml(file) as Record<string, unknown>;

		const validated = validate_config(raw, {
			name: 'lightning',
			keys: {
				error_url: { type: 'string', required: false },
				prefix: { type: 'string', required: false },
			},
		}) as Omit<config, 'plugins'> & { plugins: cli_plugin[] };

		if (
			!('type' in validated.database) ||
			typeof validated.database.type !== 'string' ||
			!('config' in validated.database) ||
			validated.database.config === null ||
			(validated.database.type === 'postgres' &&
				typeof validated.database.config !== 'string') ||
			(validated.database.type === 'redis' &&
				(typeof validated.database.config !== 'object' ||
					validated.database.config === null))
		) {
			return log_error('your config has an invalid `database` field', {
				without_cause: true,
			});
		}

		if (
			!validated.plugins.every(
				(p): p is cli_plugin =>
					typeof p.plugin === 'string' &&
					typeof p.config === 'object' &&
					p.config !== null,
			)
		) {
			return log_error('your config has an invalid `plugins` field', {
				without_cause: true,
			});
		}

		const plugins = [];

		for (const plugin of validated.plugins) {
			plugins.push({
				module: await import(plugin.plugin),
				config: plugin.config,
			});
		}

		set_env('LIGHTNING_ERROR_WEBHOOK', validated.error_url ?? '');

		return { ...validated, plugins };
	} catch (e) {
		log_error(e, {
			message:
				`could not open or parse your \`lightning.toml\` file at ${path}`,
			without_cause: true,
		});
	}
}
