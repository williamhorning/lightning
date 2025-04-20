# @lightning/discord

[![JSR](https://jsr.io/badges/@lightning/discord)](https://jsr.io/@lightning/discord)

@lightning/discord adds support for Discord to Lightning. To use it, you'll
first need to create a Discord bot at the
[Discord Developer Portal](https://discord.com/developers/applications). After
you do that, you will need to add the following to your `lightning.toml` file:

```toml
[[plugins]]
plugin = "jsr:@lightning/discord@0.8.0"
config.token = "your_bot_token"
```
