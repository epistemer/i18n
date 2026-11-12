module github.com/epistemer/i18n/examples/basic

go 1.26.0

replace (
	github.com/epistemer/i18n => ../..
	github.com/epistemer/i18n/i18nschema => ../../i18nschema
)

tool github.com/epistemer/cfgx/cmd/cfgx

require (
	github.com/epistemer/cfgx v0.1.0
	github.com/epistemer/i18n v0.0.0-00010101000000-000000000000
	github.com/epistemer/i18n/i18nschema v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.etcd.io/bbolt v1.5.0 // indirect
	go.yaml.in/yaml/v4 v4.0.0 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
