module github.com/williamhorning/lightning

go 1.24.5

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/bwmarrin/discordgo v0.29.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.5
	github.com/lmittmann/tint v1.1.2
	github.com/oklog/ulid/v2 v2.1.1
	github.com/PaulSonOfLars/gotgbot/v2 v2.0.0-rc.32
	github.com/yuin/goldmark v1.7.12
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)

retract v0.0.0-20250621020242-b31a16a87e8a // licenses not properly included, not a proper release version
