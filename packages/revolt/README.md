# lightning-plugin-revolt

lightning-plugin-revolt is a plugin for
[lightning](https://williamhorning.eu.org/lightning) that adds support for
telegram

## example config

```toml
# lightning.toml
# ...

[[plugins]]
plugin = "jsr:@jersey/lightning-plugin-revolt@0.8.0"
config = { token = "YOUR_REVOLT_TOKEN", user_id = "YOUR_BOT_USER_ID" }

# ...
```
