import { parseArgs } from '@std/cli/parse-args';
import { join, toFileUrl } from '@std/path';
import { type config, lightning } from './lightning.ts';
import { log_error } from './structures/errors.ts';

const version = '0.8.0';
const _ = parseArgs(Deno.args);

if (_.v || _.version) {
	console.log(version);
} else if (_.h || _.help) {
	run_help();
} else if (_._[0] === 'run') {
	if (!_.config) _.config = join(Deno.cwd(), 'config.ts');

	const config = (await import(toFileUrl(_.config).toString()))
		.default as config;

	if (config?.error_url) {
		Deno.env.set('LIGHTNING_ERROR_WEBHOOK', config.error_url);
	}

	addEventListener('error', (ev) => {
		log_error(ev.error, { extra: { type: 'global error' } });
	});

	addEventListener('unhandledrejection', (ev) => {
		log_error(ev.reason, { extra: { type: 'global rejection' } });
	});

	try {
		await lightning.create(config);
	} catch (e) {
		log_error(e, { extra: { type: 'global class error' } });
	}
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
	console.log('  Options:');
	console.log('    -h, --help: display this help message');
	console.log('    -v, --version: display the version number');
	console.log('    -c, --config: the config file to use');
	console.log('  Environment Variables:');
	console.log('    LIGHTNING_ERROR_WEBHOOK: the webhook to send errors to');
}
