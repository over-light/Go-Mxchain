package txcache

import "fmt"

var errTxDuplicated = fmt.Errorf("duplicated tx")
var errInvalidCacheConfig = fmt.Errorf("invalid cache config")
var errFailedCrossTxEviction = fmt.Errorf("failed cross tx eviction")
