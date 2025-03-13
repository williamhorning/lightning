import type { telegram_config } from './mod.ts';

export function setup_file_proxy(config: telegram_config) {
	Deno.serve({
		port: config.proxy_port,
		onListen: ({ port }) => {
			console.log(`[bolt-telegram] file proxy listening on localhost:${port}`);
			console.log(`[bolt-telegram] also available at: ${config.proxy_url}`);
		},
	}, (req: Request) => {
		const { pathname } = new URL(req.url);
		return fetch(
			`https://api.telegram.org/file/bot${config.bot_token}/${
				pathname.replace('/telegram/', '')
			}`,
		);
	});
}
