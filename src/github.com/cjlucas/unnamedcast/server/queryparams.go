package main

type SortParams interface {
	SortField() string
	Desc() bool
}

type sortParams struct {
	SortBy    string `param:"sort_by"`
	SortOrder string `param:"sort_order"`
}

func (p sortParams) SortField() string {
	return p.SortBy
}

func (p sortParams) Desc() bool {
	return p.SortOrder == "desc"
}

type LimitParams interface {
	Limit() int
}

type limitParams struct {
	Lim int `param:"limit"`
}

func (p limitParams) Limit() int {
	return p.Lim
}
