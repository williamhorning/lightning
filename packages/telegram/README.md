# lightning-plugin-telegram

lightning-plugin-telegram is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
telegram (including attachments via the included file proxy)

## example config

```ts
import type { config } from 'jsr:@jersey/lightning@0.8.0';
import { telegram_plugin } from 'jsr:@jersey/lightning-plugin-telegram@0.8.0';

export default {
	plugins: [
		telegram_plugin.new({
			bot_token: 'your_token',
			plugin_port: 8080,
			plugin_url: 'https://your.domain/telegram/',
		}),
	],
} as config;
```
