# @lightning/guilded

[![JSR](https://jsr.io/badges/@lightning/guilded)](https://jsr.io/@lightning/guilded)

@lightning/guilded adds support for Guilded. To use it, you'll first need to
create a Guilded bot. After you do that, you'll need to add the following to
your `lightning.toml` file:

```toml
[[plugins]]
plugin = "jsr:@lightning/guilded@0.8.0-alpha.4"
config.token = "your_bot_token"
```
