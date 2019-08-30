// Package assetcache provides in-memory lookup table for assets data.
// The client must use the lookup table by obtaining cache pointer
// by GetAssetCache() or directly Get().  It loads the cache content
// lazily at the first lookup, and if the content needs to be reloaded,
// Load() is to be called.
//
// The asset "key" is a string that identifies an asset in different forms.
// It's either the UUID, or the symbol name specified as follows.
//
// A symbol string, which can appear in the URL query parameter, the URL path
// element, or the JSON value, can consist of one, two or three elements split
// by a colon “:”.  The first element is the symbol name local to the exchange,
// the second element refers to the exchange that the symbol is primarily listed
// in, and the third refers to the asset class name.  If the second and third
// elements are omitted, the system assumes the primary exchange and asset class
//  for the symbol string that is uniquely identified.  Likewise, if the third
// element is omitted, the asset class is assumed to be the primary asset class
// for the pair of symbol and exchange names.
package assetcache

import (
	"sync"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/gbevents"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
)

type assetCache struct {
	m                           sync.RWMutex
	assets                      []*models.Asset
	lookupByAssetID             map[string]int
	lookupBySymbol              map[string]int
	lookupBySymbolExchange      map[string]int
	lookupBySymbolExchangeClass map[string]int
}

type AssetCache interface {
	Get(string) *models.Asset
	GetByID(uuid.UUID) *models.Asset
}

var globalCache AssetCache
var once sync.Once

func GetAssetCache() AssetCache {
	once.Do(func() {
		var err error
		globalCache, err = NewAssetCache()
		if err != nil {
			panic(err)
		}
	})
	return globalCache
}

func NewAssetCache() (AssetCache, error) {
	l := &assetCache{
		assets:                      []*models.Asset{},
		lookupByAssetID:             map[string]int{},
		lookupBySymbol:              map[string]int{},
		lookupBySymbolExchange:      map[string]int{},
		lookupBySymbolExchangeClass: map[string]int{},
	}
	if err := l.Load(); err != nil {
		return nil, err
	}

	return l, nil
}

func init() {
	gbevents.RegisterFunc(func(event *gbevents.Event) {
		if event.Name == gbevents.EventAssetRefreshed {
			log.Debug("triggered assets cache refresh")
			if globalCache != nil {
				if err := globalCache.(*assetCache).Load(); err != nil {
					log.Error("failed to refresh asset cache")
					return
				}
				log.Info("asset cache is refreshed")
			}
		}
	})
}

func loadAssetsReal() ([]*models.Asset, error) {
	var assets []*models.Asset
	// TBD: should be ordered by priority for each symbol
	// so key lookup can find the "primary" entry.
	err := db.DB().Find(&assets).Error
	return assets, err
}

// LoadAssetsFunc is a function to populate the asset cache
type LoadAssetsFunc func() ([]*models.Asset, error)

// replaceable loading function for testing purpose
var loadAssets LoadAssetsFunc = loadAssetsReal

func MockLoadAssets(f LoadAssetsFunc) LoadAssetsFunc {
	old := loadAssets
	loadAssets = f
	return old
}

func (l *assetCache) Load() error {
	l.m.Lock()
	defer l.m.Unlock()

	assets, err := loadAssets()
	if err != nil {
		return err
	}

	// make sure we start fresh
	l.lookupByAssetID = map[string]int{}
	l.lookupBySymbol = map[string]int{}
	l.lookupBySymbolExchange = map[string]int{}
	l.lookupBySymbolExchangeClass = map[string]int{}

	l.assets = assets
	for i, asset := range assets {
		l.lookupByAssetID[asset.ID] = i
		key := asset.Symbol
		l.lookupBySymbol[key] = i
		key += ":" + asset.Exchange
		l.lookupBySymbolExchange[key] = i
		key += ":" + string(asset.Class)
		l.lookupBySymbolExchangeClass[key] = i
	}

	return nil
}

func (l *assetCache) Get(key string) *models.Asset {
	l.m.RLock()
	defer l.m.RUnlock()
	if idx, ok := l.lookupBySymbol[key]; ok {
		return l.assets[idx]
	}
	if idx, ok := l.lookupByAssetID[key]; ok {
		return l.assets[idx]
	}
	if idx, ok := l.lookupBySymbolExchange[key]; ok {
		return l.assets[idx]
	}
	if idx, ok := l.lookupBySymbolExchangeClass[key]; ok {
		return l.assets[idx]
	}
	return nil
}

func (l *assetCache) GetByID(id uuid.UUID) *models.Asset {
	l.m.RLock()
	defer l.m.RUnlock()
	if idx, ok := l.lookupByAssetID[id.String()]; ok {
		return l.assets[idx]
	}
	return nil
}

func Get(key string) *models.Asset {
	return GetAssetCache().Get(key)
}

func GetByID(id uuid.UUID) *models.Asset {
	return GetAssetCache().GetByID(id)
}
