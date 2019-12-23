package smarket

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	cbg "github.com/whyrusleeping/cbor-gen"
)

var log = logging.Logger("reputation")

const (
	apiTimeout             = time.Second * 5
	fullRefreshInterval    = time.Second * 30
	maxConcurrCalculations = 3
)

type StorageMarket struct {
	api api.FullNode

	lock       sync.Mutex
	miners     map[address.Address]Miner
	asks       []StorageAsk
	reputation map[address.Address]Reputation

	stopped bool
	close   chan struct{}
	closed  chan struct{}
}

type Miner struct {
	Address    address.Address
	PeerID     peer.ID
	MinerPower types.BigInt
	TotalPower types.BigInt
}

type StorageAsk struct {
	Address     address.Address
	FILGibBlock types.FIL
}

type Reputation struct {
	Address    address.Address
	GotSlashed bool
}

func New(api api.FullNode) *StorageMarket {
	m := &StorageMarket{
		api:    api,
		close:  make(chan struct{}),
		closed: make(chan struct{}),
	}
	go m.refreshState()
	return m
}

func (s *StorageMarket) GetStorageAsks(ctx context.Context, cid cid.Cid, pricePerEpoch types.FIL, durationEpoch, minProposals int) ([]StorageAsk, error) {
	var res []StorageAsk
	s.lock.Lock()
	defer s.lock.Unlock()

	for i := 0; i < len(s.asks) && i < minProposals; i++ {
		res = append(res, s.asks[i])
	}

	return res, nil
}

func (m *StorageMarket) Close() {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.stopped {
		return
	}
	close(m.close)
	<-m.closed
}

func (s *StorageMarket) refreshState() {
	defer close(s.closed)
	for {
		select {
		case <-s.close:
			return
		case <-time.After(fullRefreshInterval):
			ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
			addrs, err := s.api.StateListMiners(ctx, nil)
			if err != nil {
				//log.Errorf("getting miner list failed: %s", err)
				cancel()
				continue
			}

			rateLim := make(chan struct{}, maxConcurrCalculations)
			var wg sync.WaitGroup
			outInfo := make(chan Miner)
			outReputation := make(chan Reputation)
			for _, a := range addrs {
				wg.Add(1)
				go func(ctx context.Context, a address.Address) {
					defer wg.Done()
					rateLim <- struct{}{}
					i, err := s.fetchMinerInfo(ctx, a)
					if err != nil {
						log.Warnf("reputation calculation of %s failed: %w", a, err)
						<-rateLim
						return
					}
					r, err := s.calculateReputation(ctx, a)
					<-rateLim
					if err != nil {
						log.Warnf("reputation calculation of %s failed: %w", a, err)
						return
					}
					outInfo <- i
					outReputation <- r
				}(ctx, a)
			}
			go func() {
				wg.Wait()
				close(outInfo)
				close(outReputation)
			}()
			miners := make(map[address.Address]Miner)
			for i := range outInfo {
				miners[i.Address] = i
			}
			reputation := make(map[address.Address]Reputation)
			for r := range outReputation {
				reputation[r.Address] = r
			}

			s.lock.Lock()
			s.miners = miners
			s.lock.Unlock()
			cancel()
		}
	}
}

func (s *StorageMarket) fetchMinerInfo(ctx context.Context, a address.Address) (Miner, error) {
	mp, err := s.api.StateMinerPower(ctx, a, nil)
	if err != nil {
		return Miner{}, nil
	}

	peerID, err := s.api.StateMinerPeerID(ctx, a, nil)
	if err != nil {
		return Miner{}, fmt.Errorf("getting miner peerID %s failed: %v", peerID.Pretty(), err)
	}
	return Miner{
		Address:    a,
		PeerID:     peerID,
		MinerPower: mp.MinerPower,
		TotalPower: mp.TotalPower,
	}, nil
}

func (s *StorageMarket) calculateReputation(ctx context.Context, a address.Address) (Reputation, error) {
	res, err := s.api.StateCall(ctx, &types.Message{
		To:     a,
		From:   a,
		Method: actors.MAMethods.IsSlashed,
	}, nil)
	if err != nil {
		return Reputation{}, fmt.Errorf("statecall to IsSlashed failed with: %v", err)
	}
	if res.ExitCode != 0 {
		return Reputation{}, fmt.Errorf("call to IsSlashed failed with exitcode: %d", res.ExitCode)
	}

	return Reputation{
		Address:    a,
		GotSlashed: bytes.Equal(res.Return, cbg.CborBoolTrue),
	}, nil
}
