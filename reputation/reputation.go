package reputation

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/filecoin/index/ask"
	"github.com/textileio/filecoin/index/miner"
	"github.com/textileio/filecoin/index/slashing"
	"github.com/textileio/filecoin/reputation/internal/source"
)

var (
	updateSourcesInterval = time.Second * 90
	log                   = logging.Logger("reputation")
)

// ReputationModule consolidates different sources of information to create a
// reputation rank of FC miners
type ReputationModule struct {
	ds      datastore.TxnDatastore
	sources *source.Store

	mi *miner.MinerIndex

	si *slashing.SlashingIndex

	ai *ask.AskIndex

	lockIndex sync.Mutex
	mIndex    miner.Index
	sIndex    slashing.Index
	aIndex    ask.Index

	lockScores sync.Mutex
	rebuild    chan struct{}
	scores     []MinerScore

	ctx    context.Context
	cancel context.CancelFunc
}

// MinerScore contains a score for a miner
type MinerScore struct {
	Addr  string
	Score int
}

// New returns a new ReputationModule
func New(ds datastore.TxnDatastore, mi *miner.MinerIndex, si *slashing.SlashingIndex, ai *ask.AskIndex) *ReputationModule {
	ctx, cancel := context.WithCancel(context.Background())
	rm := &ReputationModule{
		ds: ds,
		mi: mi,
		si: si,
		ai: ai,

		rebuild: make(chan struct{}, 1),
		ctx:     ctx,
		cancel:  cancel,
		sources: source.NewStore(ds),
	}

	go rm.updateSources()
	go rm.subscribeIndexes()
	go rm.indexBuilder()

	return rm
}

// AddSource adds a new external Source to be considered for reputation generation
func (rm *ReputationModule) AddSource(id string, maddr ma.Multiaddr) error {
	return rm.sources.Add(source.Source{ID: id, Maddr: maddr})
}

// GetTopMiners gets the top n miners with best score
func (rm *ReputationModule) GetTopMiners(n int) ([]MinerScore, error) {
	mr := make([]MinerScore, 0, n)
	rm.lockScores.Lock()
	for i := 0; i < n; i++ {
		mr = append(mr, rm.scores[i])
	}
	rm.lockScores.Unlock()
	return mr, nil
}

// Close closes the ReputationModule
func (rm *ReputationModule) Close() error {
	rm.cancel()
	return nil
}

// subscribeIndexes listen to all sources changes to trigger score regeneration
func (rm *ReputationModule) subscribeIndexes() {
	subMi := rm.mi.Listen()
	subSi := rm.si.Listen()
	subAi := rm.ai.Listen()

	for {
		rm.lockIndex.Lock()
		select {
		case <-rm.ctx.Done():
			log.Info("terminating background index update")
			rm.lockIndex.Unlock()
			return
		case <-subMi:
			rm.mIndex = rm.mi.Get()
		case <-subSi:
			rm.sIndex = rm.si.Get()
		case <-subAi:
			rm.aIndex = rm.ai.Get()
		}
		rm.lockIndex.Unlock()
		select {
		case rm.rebuild <- struct{}{}:
		default:
		}
	}
}

// indexBuilder regenerates score information from all known sources
func (rm *ReputationModule) indexBuilder() {
	for range rm.rebuild {
		log.Debug("rebuilding index")
		start := time.Now()

		sources, err := rm.sources.GetAll()
		if err != nil {
			log.Errorf("error when getting sources: %s", err)
			return
		}
		rm.lockIndex.Lock()
		minerIndex := rm.mIndex
		slashIndex := rm.sIndex
		askIndex := rm.aIndex
		rm.lockIndex.Unlock()

		scores := make([]MinerScore, 0, len(minerIndex.Chain.Power))
		for addr := range minerIndex.Chain.Power {
			score := calculateScore(addr, minerIndex, slashIndex, askIndex, sources)
			scores = append(scores, score)
		}
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].Score > scores[j].Score
		})

		rm.lockScores.Lock()
		rm.scores = scores
		rm.lockScores.Unlock()

		log.Debug("index rebuilt int %dms", time.Since(start).Milliseconds())
	}
}

// calculateScore calculates the score for a miner
func calculateScore(addr string, mi miner.Index, si slashing.Index, ai ask.Index, ss []source.Source) MinerScore {
	power := mi.Chain.Power[addr]
	powerScore := power.Relative

	var slashScore float64
	if slashes, ok := si.Miners[addr]; ok {
		slashScore = 1 / math.Pow(2, float64(len(slashes.Epochs)))
	}

	var externalScore float64
	for _, s := range ss {
		score, exist := s.Scores[addr]
		if !exist {
			continue
		}
		externalScore = s.Weight * float64(score)
	}

	var askScore float64
	if a, ok := ai.Storage[addr]; ok && a.Price < ai.StorageMedianPrice {
		askScore = 1
	}

	score := 50*slashScore + 20*powerScore + 20*externalScore + 10*askScore
	return MinerScore{
		Addr:  addr,
		Score: int(score),
	}
}

func (rm *ReputationModule) updateSources() {
	for {
		select {
		case <-rm.ctx.Done():
			log.Info("terminating background sources update")
			return
		case <-time.After(updateSourcesInterval):
			sources, err := rm.sources.GetAll()
			if err != nil {
				log.Errorf("error getting all sources from store: %s", err)
				continue
			}
			var wg sync.WaitGroup
			wg.Add(len(sources))
			for _, s := range sources {
				go func(s source.Source) {
					defer wg.Done()
					if err := s.Refresh(rm.ctx); err != nil {
						log.Error("error refreshing source %s: %s", s.ID, err)
						return
					}
					if err := rm.sources.Update(s); err != nil {
						log.Error("error persisting updated source %s: %s", s.ID, err)
						return
					}
				}(s)
			}
			wg.Wait()
			log.Debug("all sources refreshed successfully")
		}
	}
}
