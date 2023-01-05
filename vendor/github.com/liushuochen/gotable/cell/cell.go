package cell

type Cell interface {
	String() string
	Length() int
	Original() string
}
