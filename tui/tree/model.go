package tree

type Kind int

const (
	ItemMonth Kind = iota
	ItemSession
)

type Item struct {
	Kind   Kind
	Label  string
	Month  string
	Index  int
	Indent int
	// HostMissing marks sessions whose host path does not exist on local filesystem.
	HostMissing bool
}
