import { setEnv } from '@cross/env';
import { readTextFile } from '@std/fs/unstable-read-text-file';
import { parse as parse_toml } from '@std/toml';
import type { core_config } from './core.ts';
import type { database_config } from './database/mod.ts';
import { log_error } from './structures/errors.ts';

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

		if (
			!('database' in raw) ||
			typeof raw.database !== 'object' ||
			raw.database === null ||
			!('type' in raw.database) ||
			typeof raw.database.type !== 'string' ||
			!('config' in raw.database) ||
			raw.database.config === null ||
			(raw.database.type === 'postgres' &&
				typeof raw.database.config !== 'string') ||
			(raw.database.type === 'redis' &&
				(typeof raw.database.config !== 'object' ||
					raw.database.config === null))
		) {
			return log_error('your config has an invalid `database` field', {
				without_cause: true,
			});
		}

		if (
			!('plugins' in raw) ||
			!Array.isArray(raw.plugins) ||
			!raw.plugins.every(
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

		if ('error_url' in raw && typeof raw.error_url !== 'string') {
			return log_error('the `error_url` field is not a valid string', {
				without_cause: true,
			});
		}

		if ('prefix' in raw && typeof raw.prefix !== 'string') {
			return log_error('the `prefix` field is not a valid string', {
				without_cause: true,
			});
		}

		const validated = raw as unknown as config & { plugins: cli_plugin[] };

		const plugins = [];

		for (const plugin of validated.plugins) {
			plugins.push({
				module: await import(plugin.plugin),
				config: plugin.config,
			});
		}

		setEnv('LIGHTNING_ERROR_WEBHOOK', validated.error_url ?? '');

		return { ...validated, plugins };
	} catch (e) {
		log_error(e, {
			message:
				`could not open or parse your \`lightning.toml\` file at ${path}`,
			without_cause: true,
		});
	}
}
