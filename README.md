# contest

contest is the simulation tool for mitum and it's children. This repository is
for fixed network. For public network, visit
https://github.com/spikeekips/contest .

[![CircleCI](https://img.shields.io/circleci/project/github/spikeekips/contest-fixed-network/main.svg?style=flat-square&logo=circleci&label=circleci&cacheSeconds=60)](https://circleci.com/gh/spikeekips/contest-fixed-network/tree/main)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/spikeekips/contest-fixed-network?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/spikeekips/contest-fixed-network)](https://goreportcard.com/report/github.com/spikeekips/contest-fixed-network)
[![codecov](https://codecov.io/gh/spikeekips/contest-fixed-network/branch/master/graph/badge.svg)](https://codecov.io/gh/spikeekips/contest-fixed-network)
[![](http://tokei.rs/b1/github/spikeekips/contest-fixed-network?category=lines)](https://github.com/spikeekips/contest-fixed-network)

# Install

```sh
$ git clone https://github.com/spikeekips/contest-fixed-network
$ cd contest-fixed-network
$ go build -o ./contest
```

# Run

* Before running contest, you need to build mitum or mitum variants(ex. [mitum-currency](https://github.com/spikeekips/mitum-currency).
* Before running contest, check contest help, `$ contest --help`
* By default, contest looks for local mongodb(`mongodb://localhost:27017`)

```sh
$ ./contest --log-level debug --exit-after 2m ./mitum-currency ./scenario/standalone-run-with-init.yml
```

* You can find some example scenarios for contest at `./scenario` in this repository.
