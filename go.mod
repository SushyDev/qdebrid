module qdebrid

go 1.23.2

replace github.com/sushydev/real_debrid_go => ../real_debrid_go

require (
	github.com/sushydev/real_debrid_go v1.0.1
	go.uber.org/zap v1.26.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/golang-queue/queue v0.3.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)
