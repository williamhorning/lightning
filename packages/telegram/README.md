# lightning-plugin-telegram

lightning-plugin-telegram is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
telegram (including attachments via the included file proxy)

## example config

```toml
# lightning.toml
# ...

[[plugins]]
plugin = "jsr:@jersey/lightning-plugin-telegram@0.8.0"
config.token = "YOUR_TELEGRAM_TOKEN"
config.proxy_port = 9090
config.proxy_url = "http://localhost:9090"

# ...
```
