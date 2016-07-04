package db

type Query struct {
	Filter         M
	SortField      string
	SortDesc       bool
	SelectedFields []string
	OmittedFields  []string
	Limit          int
}
