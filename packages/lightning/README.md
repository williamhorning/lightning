![lightning](https://raw.githubusercontent.com/williamhorning/lightning/refs/heads/develop/logo.svg)

# @lightning/lightning

lightning is a typescript-based chatbot that supports bridging multiple chat
apps via plugins

## [docs](https://williamhorning.eu.org/lightning)

## `lightning.toml` example

```toml
prefix = "!"

[database]
type = "postgres"
config = "postgresql://server:password@postgres:5432/lightning"

[[plugins]]
plugin = "jsr:@lightning/discord@0.8.0-alpha.4"
config.token = "your_token"

[[plugins]]
plugin = "jsr:@lightning/revolt@0.8.0-alpha.4"
config.token = "your_token"
config.user_id = "your_bot_user_id"
```
