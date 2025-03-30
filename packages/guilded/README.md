# lightning-plugin-guilded

lightning-plugin-guilded is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
guilded

## example config

```ts
import { guilded_plugin } from 'jsr:@jersey/lightning-plugin-guilded@0.8.0';

export default {
	plugins: [
		guilded_plugin.new({
			token: 'your_token',
		}),
	],
};
```
