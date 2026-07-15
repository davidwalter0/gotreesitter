module github.com/davidwalter0/gotreesitter/cgo_harness

go 1.25.0

require (
	github.com/davidwalter0/gotreesitter v0.1.0
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/tree-sitter/go-tree-sitter v0.25.0
)

require github.com/mattn/go-pointer v0.0.1 // indirect

replace github.com/davidwalter0/gotreesitter => ..
