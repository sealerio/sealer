package installer

type executor interface {
	install() error
}
