package update

import "errors"

var ErrNilMiniBlocksStorage = errors.New("nil miniBlocks storage")

var ErrUnknownType = errors.New("unknown type")

var ErrNilStateSyncer = errors.New("nil state syncer")

var ErrNoFileToImport = errors.New("no files to import")

var ErrEndOfFile = errors.New("end of file")

var ErrHashMissmatch = errors.New("hash missmatch")

var ErrNilDataWriter = errors.New("nil data writer")

var ErrNilDataReader = errors.New("nil data reader")

var ErrInvalidFolderName = errors.New("invalid folder name")

var ErrNilStorage = errors.New("nil storage")
