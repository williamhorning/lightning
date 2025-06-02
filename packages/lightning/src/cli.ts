import { setup_bridge } from './bridge/setup.ts';
import { parse_config } from './cli_config.ts';
import { core } from './core.ts';
import { handle_migration } from './database/mod.ts';
import { cwd, exit, get_args } from './structures/cross.ts';
import { LightningError } from './structures/errors.ts';

/**
 * This module provides the Lightning CLI, which you can use to run the bot
 * @module
 */

const args = get_args();

if (args[0] === 'migrate') {
	handle_migration();
} else if (args[0] === 'run') {
	try {
		const config = await parse_config(
			new URL(args[1] ?? 'lightning.toml', `file://${cwd()}/`),
		);
		const lightning = new core(config);
		await setup_bridge(lightning, config.database);
	} catch (e) {
		await new LightningError(e, {
			extra: { type: 'global class error' },
			without_cause: true,
		}).log();

		exit(1);
	}
} else if (args[0] === 'version') {
	console.log('0.8.0-alpha.4');
} else {
	console.log(
		`lightning v0.8.0-alpha.4 - extensible chatbot connecting communities`,
	);
	console.log('  Usage: lightning [subcommand]');
	console.log('  Subcommands:');
	console.log('    run <config>: run a lightning instance');
	console.log('    migrate: migrate databases');
	console.log('    version: display the version number');
	console.log('    help: display this help message');
	console.log('  Environment Variables:');
	console.log('    LIGHTNING_ERROR_WEBHOOK: the webhook to send errors to');
	console.log('    LIGHTNING_MIGRATE_CONFIRM: confirm migration on startup');
}
