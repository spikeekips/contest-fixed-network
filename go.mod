module github.com/spikeekips/contest-fixed-network

go 1.16

require (
	github.com/alecthomas/kong v0.2.20
	github.com/docker/docker v23.0.3+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/hpcloud/tail v1.0.0
	github.com/oklog/ulid v1.3.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.0
	github.com/spikeekips/contest v0.0.0-00010101000000-000000000000
	github.com/spikeekips/mitum v0.0.0-20230719201241-970c309debef
	github.com/stretchr/testify v1.8.2
	go.mongodb.org/mongo-driver v1.8.0
	go.uber.org/automaxprocs v1.5.1
	golang.org/x/crypto v0.1.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/spikeekips/contest => ./

replace github.com/spikeekips/mitum => github.com/spikeekips/mitum-fixed-network v0.0.0-20230719202330-5a6e01ea13b3
