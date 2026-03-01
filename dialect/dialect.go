package dialect

// Dialect represents the different flavors of SQL based on the currently-used
// database drivers. All implementations of this interface should return the
// concrete queries that will be used by the migrator for house-keeping work
type Dialect interface {
	CreateMetadataTable(name string) string
	InsertMetadata(name string) string
	GetChecksums(name string) string
	TableExists() string
}
