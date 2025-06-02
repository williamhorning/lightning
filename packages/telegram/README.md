# @lightning/telegram

[![JSR](https://jsr.io/badges/@lightning/telegram)](https://jsr.io/@lightning/telegram)

@lightning/telegram adds support for Telegram. Before using it, you'll need to
talk with @BotFather to create a bot. After that, you need to add the following
to your config:

```toml
[[plugins]]
plugin = "jsr:@lightning/telegram@0.8.0-alpha.4"
config.token = "your_bot_token"
config.proxy_port = 9090
config.proxy_url = "https://example.com:9090"
```

Additionally, you will need to expose the port provided at the URL provided for
attachments sent from Telegram to work properly
