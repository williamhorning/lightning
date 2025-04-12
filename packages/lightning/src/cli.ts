import { parseArgs } from '@std/cli/parse-args';
import { join } from '@std/path';
import { parse_config } from './cli_config.ts';
import { log_error } from './structures/errors.ts';
import { handle_migration } from './database/mod.ts';
import { core } from './core.ts';
import { setup_bridge } from './bridge/setup.ts';

const version = '0.8.0';
const _ = parseArgs(Deno.args);

if (_.v || _.version) {
	console.log(version);
} else if (_.h || _.help) {
	run_help();
} else if (_._[0] === 'run') {
	if (!_.config) _.config = join(Deno.cwd(), 'lightning.toml');

	try {
		const config = await parse_config(_.config);
		const lightning = new core(config);
		setup_bridge(lightning, config.database);
	} catch (e) {
		log_error(e, { extra: { type: 'global class error' } });
	}
} else if (_._[0] === 'migrate') {
	handle_migration();
} else {
	console.log('[lightning] command not found, showing help');
	run_help();
}

function run_help() {
	console.log(
		`lightning v${version} - extensible chatbot connecting communities`,
	);
	console.log('  Usage: lightning [subcommand] <options>');
	console.log('  Subcommands:');
	console.log('    run: run a lightning instance');
	console.log('    migrate: migrate databases');
	console.log('  Options:');
	console.log('    -h, --help: display this help message');
	console.log('    -v, --version: display the version number');
	console.log('    -c, --config: the config file to use');
	console.log('  Environment Variables:');
	console.log('    LIGHTNING_ERROR_WEBHOOK: the webhook to send errors to');
	console.log('    LIGHTNING_MIGRATE_CONFIRM: confirm migration on startup');
}
