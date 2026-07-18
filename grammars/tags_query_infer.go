package grammars

import (
	"strings"
	"sync"
)

type tagsQueryPattern struct {
	Query   string
	Symbols []string
}

var (
	inferredTagsQueryCache sync.Map // map[string]string
)

var inferredTagsQueryOverrides = map[string]string{
	"go": strings.Join([]string{
		"(function_declaration name: (identifier) @name) @definition.function",
		"(method_declaration name: (field_identifier) @name) @definition.method",
		"(call_expression function: (identifier) @name) @reference.call",
		"(call_expression function: (selector_expression field: (field_identifier) @name)) @reference.call",
	}, "\n"),

	// python: starts from the same generic-table-derived lines this
	// language previously resolved to (function/class/call), plus explicit
	// module-level assignment capture. Two anchor shapes are listed because
	// this fork elides the expression_statement supertype and hoists a bare
	// `assignment` directly under `module`, while a grammar build that
	// preserves the supertype wraps it in expression_statement first; only
	// the second pattern currently matches this fork's trees, but the first
	// keeps the override portable. Anchoring to `(module ...)` (a direct
	// child of the file root) is what excludes function-local assignments
	// — an assignment nested under `function_definition -> block` is never
	// a direct child of `module`, so it can't match either pattern.
	"python": strings.Join([]string{
		"(function_definition (identifier) @name) @definition.function",
		"(class_definition (identifier) @name) @definition.class",
		"(call (identifier) @name) @reference.call",
		"(call (attribute (identifier) @name)) @reference.call",
		"(module (expression_statement (assignment left: (identifier) @name) @definition.constant))",
		"(module (assignment left: (identifier) @name) @definition.constant)",
	}, "\n"),

	// javascript: starts from the same generic-table-derived lines this
	// language previously resolved to (function/method/class/call), plus
	// top-level const/let coverage. The Tagger has no match-precedence or
	// dedupe (grammars.ResolveTagsQuery -> gotreesitter.Tagger.tagTree emits
	// one Tag per QueryMatch, unconditionally) — so a naive catch-all
	// `(variable_declarator name: (identifier) @name)` pattern would
	// double-tag every arrow/function-valued declarator as BOTH
	// definition.function and definition.constant. To keep the two kinds
	// mutually exclusive by construction (not by relying on any
	// pick-first/pick-last consumer behavior), the constant pattern uses an
	// explicit value-type allowlist that deliberately omits
	// arrow_function/function_expression.
	"javascript": strings.Join([]string{
		"(function_declaration (identifier) @name) @definition.function",
		"(method_definition (property_identifier) @name) @definition.method",
		"(method_definition (identifier) @name) @definition.method",
		"(class_declaration (identifier) @name) @definition.class",
		"(call_expression (identifier) @name) @reference.call",
		"(call_expression (property_identifier) @name) @reference.call",
		"(call_expression (member_expression (property_identifier) @name)) @reference.call",
		"(lexical_declaration (variable_declarator name: (identifier) @name value: (arrow_function))) @definition.function",
		"(lexical_declaration (variable_declarator name: (identifier) @name value: (function_expression))) @definition.function",
		"(lexical_declaration (variable_declarator name: (identifier) @name value: [(number) (string) (template_string) (true) (false) (null) (undefined) (object) (array) (call_expression) (member_expression) (new_expression) (binary_expression) (unary_expression) (identifier) (regex) (ternary_expression)])) @definition.constant",
	}, "\n"),

	// rust: starts from the same generic-table-derived lines this language
	// previously resolved to (function/struct/enum/trait/call/macro kept
	// as-is), plus const/static item capture. const_item and static_item
	// are distinct node types from struct_item/enum_item/trait_item, so
	// there is no overlap risk with the existing @definition.type patterns.
	"rust": strings.Join([]string{
		"(function_item (identifier) @name) @definition.function",
		"(function_signature_item (identifier) @name) @definition.function",
		"(struct_item (type_identifier) @name) @definition.type",
		"(enum_item (type_identifier) @name) @definition.type",
		"(trait_item (type_identifier) @name) @definition.type",
		"(call_expression (identifier) @name) @reference.call",
		"(call_expression (field_identifier) @name) @reference.call",
		"(call_expression (scoped_identifier (identifier) @name)) @reference.call",
		"(macro_invocation (identifier) @name) @reference.call",
		"(const_item (identifier) @name) @definition.constant",
		"(static_item (identifier) @name) @definition.constant",
	}, "\n"),

	// c: starts from the same generic-table-derived lines this language
	// previously resolved to (function/type/struct/call), plus #define and
	// file-scope variable capture. preproc_def uses the `name:` field so it
	// never touches preproc_function_def (a distinct node type for
	// function-like macros, so no overlap there). The two
	// definition.variable patterns are anchored to `(translation_unit
	// (declaration ...))` — declaration must be a DIRECT child of the file
	// root — which is what excludes function-local declarations (a local
	// `declaration` inside a function body is nested under
	// `function_definition -> compound_statement`, never a direct child of
	// translation_unit). Both patterns use the `declarator:` field
	// explicitly (not a bare positional `(identifier)`) so that, for e.g.
	// `int a = b;`, only the declared name `a` is captured and not the
	// initializer-side reference `b` (which the grammar also types as
	// `identifier` but under the `value:` field). Declarators more complex
	// than a bare identifier (pointer/array declarators, e.g. `int *p` or
	// `int arr[3]`) are intentionally NOT matched here — extending to those
	// shapes was judged higher over-capture risk than benefit for a first
	// pass; the plain-identifier declarator case is common and unambiguous.
	"c": strings.Join([]string{
		"(function_definition (identifier) @name) @definition.function",
		"(function_definition (field_identifier) @name) @definition.function",
		"(function_definition (function_declarator (identifier) @name)) @definition.function",
		"(function_definition (function_declarator (field_identifier) @name)) @definition.function",
		"(type_definition (type_identifier) @name) @definition.type",
		"(type_definition (identifier) @name) @definition.type",
		"(struct_specifier (type_identifier) @name) @definition.type",
		"(call_expression (identifier) @name) @reference.call",
		"(call_expression (field_identifier) @name) @reference.call",
		"(preproc_def name: (identifier) @name) @definition.constant",
		"(translation_unit (declaration (init_declarator declarator: (identifier) @name)) @definition.variable)",
		"(translation_unit (declaration declarator: (identifier) @name) @definition.variable)",
	}, "\n"),
}

var inferredTagsQueryPatterns = []tagsQueryPattern{
	// Common definitions.
	{Query: "(function_declaration (identifier) @name) @definition.function", Symbols: []string{"function_declaration", "identifier"}},
	{Query: "(function_declaration (type_identifier) @name) @definition.function", Symbols: []string{"function_declaration", "type_identifier"}},
	{Query: "(function_definition (identifier) @name) @definition.function", Symbols: []string{"function_definition", "identifier"}},
	{Query: "(function_definition (field_identifier) @name) @definition.function", Symbols: []string{"function_definition", "field_identifier"}},
	{Query: "(function_definition (function_declarator (identifier) @name)) @definition.function", Symbols: []string{"function_definition", "function_declarator", "identifier"}},
	{Query: "(function_definition (function_declarator (field_identifier) @name)) @definition.function", Symbols: []string{"function_definition", "function_declarator", "field_identifier"}},
	{Query: "(method_declaration (identifier) @name) @definition.method", Symbols: []string{"method_declaration", "identifier"}},
	{Query: "(method_declaration (field_identifier) @name) @definition.method", Symbols: []string{"method_declaration", "field_identifier"}},
	{Query: "(method_definition (property_identifier) @name) @definition.method", Symbols: []string{"method_definition", "property_identifier"}},
	{Query: "(method_definition (identifier) @name) @definition.method", Symbols: []string{"method_definition", "identifier"}},
	{Query: "(class_declaration (identifier) @name) @definition.class", Symbols: []string{"class_declaration", "identifier"}},
	{Query: "(class_declaration (type_identifier) @name) @definition.class", Symbols: []string{"class_declaration", "type_identifier"}},
	{Query: "(class_definition (identifier) @name) @definition.class", Symbols: []string{"class_definition", "identifier"}},
	{Query: "(interface_declaration (identifier) @name) @definition.interface", Symbols: []string{"interface_declaration", "identifier"}},
	{Query: "(interface_declaration (type_identifier) @name) @definition.interface", Symbols: []string{"interface_declaration", "type_identifier"}},
	{Query: "(enum_declaration (identifier) @name) @definition.type", Symbols: []string{"enum_declaration", "identifier"}},
	{Query: "(enum_declaration (type_identifier) @name) @definition.type", Symbols: []string{"enum_declaration", "type_identifier"}},
	{Query: "(constructor_declaration (identifier) @name) @definition.constructor", Symbols: []string{"constructor_declaration", "identifier"}},
	{Query: "(type_definition (type_identifier) @name) @definition.type", Symbols: []string{"type_definition", "type_identifier"}},
	{Query: "(type_definition (identifier) @name) @definition.type", Symbols: []string{"type_definition", "identifier"}},
	{Query: "(type_declaration (type_spec (type_identifier) @name)) @definition.type", Symbols: []string{"type_declaration", "type_spec", "type_identifier"}},
	{Query: "(type_declaration (type_alias (type_identifier) @name)) @definition.type", Symbols: []string{"type_declaration", "type_alias", "type_identifier"}},
	{Query: "(function_item (identifier) @name) @definition.function", Symbols: []string{"function_item", "identifier"}},
	{Query: "(function_signature_item (identifier) @name) @definition.function", Symbols: []string{"function_signature_item", "identifier"}},
	{Query: "(struct_item (type_identifier) @name) @definition.type", Symbols: []string{"struct_item", "type_identifier"}},
	{Query: "(enum_item (type_identifier) @name) @definition.type", Symbols: []string{"enum_item", "type_identifier"}},
	{Query: "(trait_item (type_identifier) @name) @definition.type", Symbols: []string{"trait_item", "type_identifier"}},
	{Query: "(class_specifier (type_identifier) @name) @definition.class", Symbols: []string{"class_specifier", "type_identifier"}},
	{Query: "(struct_specifier (type_identifier) @name) @definition.type", Symbols: []string{"struct_specifier", "type_identifier"}},

	// Constants and variables.
	{Query: "(const_spec (identifier) @name) @definition.constant", Symbols: []string{"const_spec", "identifier"}},
	{Query: "(var_spec (identifier) @name) @definition.variable", Symbols: []string{"var_spec", "identifier"}},
	{Query: "(short_var_declaration (identifier) @name) @definition.variable", Symbols: []string{"short_var_declaration", "identifier"}},

	// Common call references.
	{Query: "(call_expression (identifier) @name) @reference.call", Symbols: []string{"call_expression", "identifier"}},
	{Query: "(call_expression (field_identifier) @name) @reference.call", Symbols: []string{"call_expression", "field_identifier"}},
	{Query: "(call_expression (property_identifier) @name) @reference.call", Symbols: []string{"call_expression", "property_identifier"}},
	{Query: "(call_expression (member_expression (property_identifier) @name)) @reference.call", Symbols: []string{"call_expression", "member_expression", "property_identifier"}},
	{Query: "(call_expression (selector_expression (field_identifier) @name)) @reference.call", Symbols: []string{"call_expression", "selector_expression", "field_identifier"}},
	{Query: "(call_expression (scoped_identifier (identifier) @name)) @reference.call", Symbols: []string{"call_expression", "scoped_identifier", "identifier"}},
	{Query: "(call (identifier) @name) @reference.call", Symbols: []string{"call", "identifier"}},
	{Query: "(call (attribute (identifier) @name)) @reference.call", Symbols: []string{"call", "attribute", "identifier"}},
	{Query: "(method_invocation (identifier) @name) @reference.call", Symbols: []string{"method_invocation", "identifier"}},
	{Query: "(macro_invocation (identifier) @name) @reference.call", Symbols: []string{"macro_invocation", "identifier"}},
}

func inferredTagsQuery(entry LangEntry) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		return ""
	}

	if cached, ok := inferredTagsQueryCache.Load(name); ok {
		return cached.(string)
	}

	if query := inferredTagsQueryOverrides[name]; strings.TrimSpace(query) != "" {
		inferredTagsQueryCache.Store(name, query)
		return query
	}

	if entry.Language == nil {
		inferredTagsQueryCache.Store(name, "")
		return ""
	}
	lang := entry.Language()
	if lang == nil {
		inferredTagsQueryCache.Store(name, "")
		return ""
	}

	hasSymbol := func(symbol string) bool {
		_, ok := lang.SymbolByName(symbol)
		return ok
	}

	lines := make([]string, 0, 32)
	seen := make(map[string]struct{}, len(inferredTagsQueryPatterns))
	for _, pattern := range inferredTagsQueryPatterns {
		missing := false
		for _, symbol := range pattern.Symbols {
			if !hasSymbol(symbol) {
				missing = true
				break
			}
		}
		if missing {
			continue
		}
		if _, ok := seen[pattern.Query]; ok {
			continue
		}
		seen[pattern.Query] = struct{}{}
		lines = append(lines, pattern.Query)
	}

	query := strings.Join(lines, "\n")
	inferredTagsQueryCache.Store(name, query)
	return query
}
