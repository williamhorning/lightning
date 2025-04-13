# lightning-plugin-discord

lightning-plugin-discord is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
discord

## example config

```toml
# lightning.toml
# ...

[[plugins]]
plugin = "jsr:@jersey/lightning-plugin-discord@0.8.0"
config.token = "YOUR_DISCORD_TOKEN"

# ...
```
