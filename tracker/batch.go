package tracker

import (
	"fmt"
	"sync"

	"github.com/wcgcyx/ethgen/idgen"
	"github.com/ethereum/go-ethereum/core/types"
	wr "github.com/mroth/weightedrand"
)

type BatchTracker struct {
	weight uint
	// Sub-trackers
	trackers []Tracker
}

func NewBatchTracker(trackers []Tracker) *BatchTracker {
	return &BatchTracker{
		weight:   0,
		trackers: trackers,
	}
}

func NewERC20BatchTracker(idGen idgen.IdGenerator, contractAddrs []string, maxBlocks uint) *BatchTracker {
	trackers := make([]Tracker, 0)
	for _, contractAddr := range contractAddrs {
		trackers = append(trackers, NewERC20ContractTracker(idGen, contractAddr, maxBlocks))
	}
	return &BatchTracker{
		weight:   0,
		trackers: trackers,
	}
}

func NewERC721BatchTracker(idGen idgen.IdGenerator, contractAddrs []string, maxBlocks uint) *BatchTracker {
	trackers := make([]Tracker, 0)
	for _, contractAddr := range contractAddrs {
		trackers = append(trackers, NewERC721ContractTracker(idGen, contractAddr, maxBlocks))
	}
	return &BatchTracker{
		weight:   0,
		trackers: trackers,
	}
}

func (t *BatchTracker) Trackers() []Tracker {
	return t.trackers
}

func (t *BatchTracker) ApplyBlock(blk *types.Block) error {
	wg := sync.WaitGroup{}
	for _, tracker := range t.trackers {
		wg.Add(1)
		go func(tracker Tracker) {
			defer wg.Done()
			err := tracker.ApplyBlock(blk)
			if err != nil {
				fmt.Println(err.Error())
			}
		}(tracker)
	}
	wg.Wait()
	t.weight = 0
	for _, tracker := range t.trackers {
		t.weight += tracker.CurrentWeight()
	}
	return nil
}

func (t *BatchTracker) CurrentWeight() uint {
	return t.weight
}

func (t *BatchTracker) GenerateQuery(number uint) ([]string, error) {
	counts := make([]uint, len(t.trackers))
	choices := make([]wr.Choice, 0)
	for index, tracker := range t.trackers {
		choices = append(choices, wr.NewChoice(index, tracker.CurrentWeight()))
	}
	contractChooser, err := wr.NewChooser(choices...)
	if err != nil {
		return nil, err
	}
	for i := uint(0); i < number; i++ {
		contract := contractChooser.Pick().(int)
		counts[contract]++
	}
	resList := make([][]string, len(t.trackers))
	wg := sync.WaitGroup{}
	for index, tracker := range t.trackers {
		wg.Add(1)
		go func(index int, tracker Tracker) {
			defer wg.Done()
			count := counts[index]
			res, err := tracker.GenerateQuery(count)
			if err != nil {
				fmt.Println(err.Error())
			} else {
				resList[index] = res
			}
		}(index, tracker)
	}
	wg.Wait()
	res := make([]string, 0)
	for _, sub := range resList {
		res = append(res, sub...)
	}
	return res, nil
}

func (t *BatchTracker) Status() string {
	res := fmt.Sprintf("%v - ", t.weight)
	for _, tracker := range t.trackers {
		res += tracker.Status() + " "
	}
	return res
}
