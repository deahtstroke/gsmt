package gsmt

type Dialect interface {
	Placeholder(i int) string
	GetMetadataTables() map[string]string
}
