# @lightning/revolt

[![JSR](https://jsr.io/badges/@lightning/revolt)](https://jsr.io/@lightning/revolt)

@lightning/telegram adds support for Revolt. To use it, you'll need to create a
Revolt bot first. After that, you need to add the following to your config file:

```toml
[[plugins]]
plugin = "jsr:@lightning/revolt@0.8.0-alpha.2"
config.token = "your_bot_token"
config.user_id = "your_bot_user_id"
```
