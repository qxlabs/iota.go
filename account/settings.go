package account

import (
	"github.com/iotaledger/iota.go/account/event"
	"github.com/iotaledger/iota.go/account/store"
	"github.com/iotaledger/iota.go/account/store/inmemory"
	"github.com/iotaledger/iota.go/account/timesrc"
	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/trinary"
	"strings"
)

// InputSelectionFunc defines a function which given the account, transfer value and the flag balance check,
// computes the inputs for fulfilling the transfer or the usable balance of the account.
// The InputSelectionFunc must obey to the rules of conditional deposit requests to ensure consistency.
// It returns the computed balance/transfer value, inputs and the key indices to remove from the store.
type InputSelectionFunc func(acc *account, transferValue uint64, balanceCheck bool) (uint64, []api.Input, []uint64, error)

// AddrGenFunc defines a function which given the index, security level and addChecksum flag, generates a new address.
type AddrGenFunc func(index uint64, secLvl consts.SecurityLevel, addChecksum bool) (trinary.Hash, error)

// PrepareTransfersFunc defines a function which prepares the transaction trytes by generating a bundle,
// filling in transfers and inputs, adding the remainder address and signing all input transactions.
type PrepareTransfersFunc func(transfers bundle.Transfers, options api.PrepareTransfersOptions) ([]trinary.Trytes, error)

func DefaultAddrGen(provider SeedProvider) AddrGenFunc {
	return func(index uint64, secLvl consts.SecurityLevel, addChecksum bool) (trinary.Hash, error) {
		seed, err := provider.Seed()
		if err != nil {
			return "", err
		}
		return address.GenerateAddress(seed, index, secLvl, addChecksum)
	}
}

func DefaultPrepareTransfers(a *api.API, provider SeedProvider) PrepareTransfersFunc {
	return func(transfers bundle.Transfers, options api.PrepareTransfersOptions) ([]trinary.Trytes, error) {
		seed, err := provider.Seed()
		if err != nil {
			return nil, err
		}
		return a.PrepareTransfers(seed, transfers, options)
	}
}

// Settings defines settings used by an account.
type Settings struct {
	API                 *api.API
	Store               store.Store
	SeedProv            SeedProvider
	MWM                 uint64
	Depth               uint64
	SecurityLevel       consts.SecurityLevel
	TimeSource          timesrc.TimeSource
	InputSelectionStrat InputSelectionFunc
	EventMachine        event.EventMachine
	Plugins             map[string]Plugin
	AddrGen             AddrGenFunc
	PrepareTransfers    PrepareTransfersFunc
}

var emptySeed = strings.Repeat("9", 81)

// DefaultSettings returns Settings initialized with default values:
// empty seed (81x "9" trytes), mwm: 14, depth: 3, security level: 2, no event machine,
// system timesrc, default input sel. strat, in-memory store, iota-api pointing to localhost,
// no transfer poller plugin, no promoter-reattacher plugin.
func DefaultSettings(setts ...Settings) *Settings {
	if len(setts) == 0 {
		iotaAPI, _ := api.ComposeAPI(api.HTTPClientSettings{})
		return &Settings{
			MWM:                 14,
			Depth:               3,
			SeedProv:            NewInMemorySeedProvider(emptySeed),
			SecurityLevel:       consts.SecurityLevelMedium,
			TimeSource:          &timesrc.SystemClock{},
			EventMachine:        &event.DiscardEventMachine{},
			API:                 iotaAPI,
			Store:               inmemory.NewInMemoryStore(),
			InputSelectionStrat: defaultInputSelection,
		}
	}
	defaultValue := func(val uint64, should uint64) uint64 {
		if val == 0 {
			return should
		}
		return val
	}
	sett := setts[0]
	if sett.SecurityLevel == 0 {
		sett.SecurityLevel = consts.SecurityLevelMedium
	}
	sett.Depth = defaultValue(sett.Depth, 3)
	sett.MWM = defaultValue(sett.MWM, 14)
	if sett.TimeSource == nil {
		sett.TimeSource = &timesrc.SystemClock{}
	}
	if sett.InputSelectionStrat == nil {
		sett.InputSelectionStrat = defaultInputSelection
	}
	if sett.API == nil {
		iotaAPI, _ := api.ComposeAPI(api.HTTPClientSettings{})
		sett.API = iotaAPI
	}
	if sett.Store == nil {
		sett.Store = inmemory.NewInMemoryStore()
	}
	return &sett
}
