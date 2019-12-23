module github.com/textileio/filecoin

go 1.13

require (
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/filecoin-project/lotus v0.1.2-0.20191217122501-ae0864f8aba0
	github.com/google/uuid v1.1.1
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.3.1
	github.com/ipfs/go-log v1.0.0
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/textileio/go-textile-core v0.0.0-20191205233641-31fc120682c9
	github.com/textileio/go-textile-threads v0.0.0-20191216175741-467514ac069b
	github.com/whyrusleeping/cbor-gen v0.0.0-20191216205031-b047b6acb3c0
	gopkg.in/src-d/go-log.v1 v1.0.1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
