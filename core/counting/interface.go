package counting

// Counts is an interface to interact with counts
type Counts interface {
	GetTotal() int64
	String() string
	IsInterfaceNil() bool
}
