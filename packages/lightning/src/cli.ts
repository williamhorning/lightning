import { setup_bridge } from './bridge/setup.ts';
import { parse_config } from './cli_config.ts';
import { core } from './core.ts';
import { handle_migration } from './database/mod.ts';
import { log_error } from './structures/errors.ts';

if (Deno.args[0] === 'migrate') {
	handle_migration();
} else if (Deno.args[0] === 'run') {
	try {
		const config = await parse_config(
			new URL(Deno.args[1] ?? 'lightning.toml', `file://${Deno.cwd()}/`),
		);
		const lightning = new core(config);
		await setup_bridge(lightning, config.database);
	} catch (e) {
		log_error(e, {
			extra: { type: 'global class error' },
			without_cause: true,
		});
	}
} else if (Deno.args[0] === 'version') {
	console.log('0.8.0');
} else {
	console.log(
		`lightning v0.8.0 - extensible chatbot connecting communities`,
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
