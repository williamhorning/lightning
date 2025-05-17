import { log_error } from './errors.ts';

/** A config schema */
export interface config_schema {
	name: string;
	keys: Record<string, {
		type: 'string' | 'number' | 'boolean';
		required: boolean;
	}>;
}

/** Validate an item based on a schema */
export function validate_config<T>(config: unknown, schema: config_schema): T {
	if (typeof config !== 'object' || config === null) {
		log_error(`[${schema.name}] config is not an object`, {
			without_cause: true,
		});
	}

	for (const [key, { type, required }] of Object.entries(schema.keys)) {
		const value = (config as Record<string, unknown>)[key];

		if (required && value === undefined) {
			log_error(`[${schema.name}] missing required config key '${key}'`, {
				without_cause: true,
			});
		} else if (value !== undefined && typeof value !== type) {
			log_error(`[${schema.name}] config key '${key}' must be a ${type}`, {
				without_cause: true,
			});
		}
	}

	return config as T;
}
