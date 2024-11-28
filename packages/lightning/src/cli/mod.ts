import { parseArgs } from '@std/cli/parse-args';
import { join, toFileUrl } from '@std/path';
import { logError } from '../errors.ts';
import { type config, lightning } from '../lightning.ts';

const version = '0.8.0';
const _ = parseArgs(Deno.args);

if (_.v || _.version) {
	console.log(version);
} else if (_.h || _.help) {
	run_help();
} else if (_._[0] === 'run') {
	// TODO(jersey): this is somewhat broken when acting as a JSR package
	if (!_.config) _.config = join(Deno.cwd(), 'config.ts');

	const config: config = (await import(toFileUrl(_.config).toString())).default;

	addEventListener('error', (ev) => {
		logError(ev.error, { extra: { type: 'global error' } });
	});

	addEventListener('unhandledrejection', (ev) => {
		logError(ev.reason, { extra: { type: 'global rejection' } });
	});

	try {
		await lightning.create(config);
	} catch (e) {
		logError(e, { extra: { type: 'global class error' } });
	}
} else if (_._[0] === 'migrations') {
	// TODO(jersey): implement migrations
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
	console.log('    migrations: run migration script');
	console.log('  Options:');
	console.log('    -h, --help: display this help message');
	console.log('    -v, --version: display the version number');
	console.log('    -c, --config: the config file to use');
	console.log('  Environment Variables:');
	console.log('    LIGHTNING_ERROR_WEBHOOK: the webhook to send errors to');
}
