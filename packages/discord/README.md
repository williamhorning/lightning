# lightning-plugin-discord

lightning-plugin-discord is a plugin for
[lightning](https://williamhroning.eu.org/lightning) that adds support for
discord

## example config

```ts
import type { config } from 'jsr:@jersey/lightning@0.7.4';
import { discord_plugin } from 'jsr:@jersey/lightning-plugin-discord@0.7.4';

export default {
	redis_host: 'localhost',
	redis_port: 6379,
	plugins: [
		discord_plugin.new({
			app_id: 'your_application_id',
			token: 'your_token',
			slash_cmds: false,
		}),
	],
} as config;
```
