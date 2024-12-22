# lightning-plugin-revolt

lightning-plugin-revolt is a plugin for
[lightning](https://williamhroning.eu.org/lightning) that adds support for
telegram

## example config

```ts
import type { config } from 'jsr:@jersey/lightning@0.7.4';
import { revolt_plugin } from 'jsr:@jersey/lightning-plugin-revolt@0.7.4';

export default {
	redis_host: 'localhost',
	redis_port: 6379,
	plugins: [
		revolt_plugin.new({
			token: 'your_token',
		}),
	],
} as config;
```
