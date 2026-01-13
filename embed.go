package tlgram

import _ "embed"

//go:embed CHANGELOG.md
var Changelog string

//go:embed config.default.toml
var DefaultConfig string
