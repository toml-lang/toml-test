module github.com/BurntSushi/toml-test

go 1.16

require (
	github.com/BurntSushi/toml v0.3.2-0.20210624061728-01bfc69d1057
	// no_term branch, which doesn't depend on x/term and x/sys
	zgo.at/zli v0.0.0-20210619044753-e7020a328e59
)

replace github.com/BurntSushi/toml => github.com/BurntSushi/toml v0.3.2-0.20210704081116-ccff24ee4463
