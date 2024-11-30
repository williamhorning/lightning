import type { telegram_config } from './mod.ts';

export function file_proxy(config: telegram_config) {
    Deno.serve({
        port: config.plugin_port,
        onListen: (addr) => {
            console.log(
                `bolt-telegram: file proxy listening on http://localhost:${addr.port}`,
                `\nbolt-telegram: also available at: ${config.plugin_url}`,
            );
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