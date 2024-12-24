# lightning-plugin-guilded

lightning-plugin-guilded is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
telegram

## example config

```ts
import type { config } from 'jsr:@jersey/lightning@0.7.4';
import { guilded_plugin } from 'jsr:@jersey/lightning-plugin-guilded@0.7.4';

export default {
	plugins: [
		guilded_plugin.new({
			token: 'your_token',
		}),
	],
} as config;
```
