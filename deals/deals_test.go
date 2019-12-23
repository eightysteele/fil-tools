package deals

import (
	"os"
	"testing"

	logging "github.com/ipfs/go-log"
	"github.com/textileio/filecoin/client"
)

func TestMain(t *testing.M) {
	logging.SetDebugLogging()
	os.Exit(t.Run())
}

func TestAskCache(t *testing.T) {
	daemonAddr := "127.0.0.1:1234"
	authToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.js4fvXtcRZqEgC0ke0GDDi8QcwG52jSJcfBmM0ARLro"
	c, cls, err := client.New(daemonAddr, authToken)
	if err != nil {
		panic("couldn't create the client")
	}
	defer cls()

	asks, err := takeFreshAskSnapshot(c)
	checkErr(t, err)
	if len(asks) == 0 {
		t.Fatalf("current asks can't be empty")
	}
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
