package config

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/oklog/ulid"
)

var entropy io.Reader

func init() {
	entropy = ulid.Monotonic(rand.Reader, 0)
}

func ULID() ulid.ULID {
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
}
