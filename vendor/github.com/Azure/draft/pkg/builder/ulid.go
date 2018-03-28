package builder

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid"
)

func getulid() string { return <-ulidc }

// A channel which returns build ulids.
var ulidc = make(chan string)

func init() {
	rnd := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	go func() {
		for {
			ulidc <- ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rnd).String()
		}
	}()
}
