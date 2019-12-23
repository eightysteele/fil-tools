package client

import (
	"context"
	"os"
	"testing"

	"github.com/filecoin-project/lotus/api"
)

var nodeAPI api.FullNode

// Auth token can be generated with: `lotus auth create-token --perm admin`
func TestMain(m *testing.M) {
	daemonAddr := "127.0.0.1:1234"
	authToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.js4fvXtcRZqEgC0ke0GDDi8QcwG52jSJcfBmM0ARLro"
	c, cls, err := New(daemonAddr, authToken)
	if err != nil {
		panic("couldn't create the client")
	}
	defer cls()
	nodeAPI = c

	os.Exit(m.Run())
}

func TestClientVersion(t *testing.T) {
	t.Parallel()
	if _, err := nodeAPI.Version(context.Background()); err != nil {
		t.Fatalf("error when getting client version: %s", err)
	}
}

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
