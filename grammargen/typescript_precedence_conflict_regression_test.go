package grammargen

import "testing"

// TestTypeScriptUnaryBinaryOverCallPrecedenceRegression pins the fix for a
// gotreesitter-side defect where a declared shift/reduce conflict between
// unary_expression/binary_expression (PrecLeft(21) "unary_void") and
// call_expression (Prec(22) "call") was left as genuine runtime GLR ambiguity
// instead of being resolved by the grammar's own precedence table — even
// though "call" and "unary_void" co-occur in the SAME precedence level and
// tree-sitter's own generator resolves shift/reduce conflicts by precedence
// regardless of whether the two sides share an LHS symbol.
//
// The bug is context-sensitive: it needs a genuinely unresolved GLR fork to
// still be open when the unary/call ambiguity is reached, so an isolated
// "!f()" statement parses correctly by luck (only one live GLR stack ever
// reaches the conflict) while the identical construct, preceded anywhere
// earlier in the same file by an unrelated declaration containing a
// TypeScript function_type ("(param: T) => U", itself a declared-conflict
// construct), produces a second, never-resolved GLR fork whose final
// selection previously fell back to a symbol-ID tie-break with no precedence
// awareness. See docs/sessions for the full root-cause writeup.
//
// Regression source: tree-sitter-typescript's own test/corpus and the
// TypeScript compiler's parser.ts (examples/parser.ts upstream), both of
// which hit this pattern pervasively via `return visitNode(...) ||
// visitNode(...)` and `if (!isNodeKind(...))`.
func TestTypeScriptUnaryBinaryOverCallPrecedenceRegression(t *testing.T) {
	if raceEnabled {
		t.Skip("skip heavyweight TypeScript parity generation under -race; non-race coverage keeps the generated-vs-reference check")
	}

	assertImportedDeepParityCases(t, "typescript", []struct {
		name string
		src  string
	}{
		{
			name: "unary_negation_over_call_after_function_type_declaration",
			src:  "let NodeConstructor: (kind: number) => Node;\n!isNodeKind(kind);\n",
		},
		{
			name: "logical_or_over_calls_after_function_type_declaration",
			src:  "let NodeConstructor: (kind: number) => Node;\na() || b();\n",
		},
		{
			// Mirrors the TypeScript compiler's real forEachChild-dispatcher
			// idiom: an exported function inside a namespace, with a
			// preceding sibling `let` declared via a construct-signature
			// type ("new (...) => T"), followed by a deeply nested
			// if/else-if chain whose final branch negates a call.
			name: "unary_negation_over_call_in_namespaced_dispatcher",
			src: "namespace ts {\n" +
				"    const enum SignatureFlags {\n" +
				"        None = 0,\n" +
				"    }\n" +
				"\n" +
				"    let NodeConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n" +
				"    let TokenConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n" +
				"    let IdentifierConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n" +
				"    let SourceFileConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n" +
				"\n" +
				"    export function createNode(kind: SyntaxKind, pos?: number, end?: number): Node {\n" +
				"        if (kind === SyntaxKind.SourceFile) {\n" +
				"            return new (SourceFileConstructor || (SourceFileConstructor = objectAllocator.getSourceFileConstructor()))(kind, pos, end);\n" +
				"        }\n" +
				"        else if (kind === SyntaxKind.Identifier) {\n" +
				"            return new (IdentifierConstructor || (IdentifierConstructor = objectAllocator.getIdentifierConstructor()))(kind, pos, end);\n" +
				"        }\n" +
				"        else if (!isNodeKind(kind)) {\n" +
				"            return new (TokenConstructor || (TokenConstructor = objectAllocator.getTokenConstructor()))(kind, pos, end);\n" +
				"        }\n" +
				"        else {\n" +
				"            return new (NodeConstructor || (NodeConstructor = objectAllocator.getNodeConstructor()))(kind, pos, end);\n" +
				"        }\n" +
				"    }\n" +
				"}\n",
		},
		{
			name: "logical_or_over_calls_in_return_statement",
			src:  "function visitNode(cbNode, node) {\n  return node && cbNode(node);\n}\nfunction f() {\n  return visitNode(cbNode, a) || visitNode(cbNode, b);\n}\n",
		},
	})
}

func TestTSXUnaryBinaryOverCallPrecedenceRegression(t *testing.T) {
	if raceEnabled {
		t.Skip("skip heavyweight TypeScript parity generation under -race; non-race coverage keeps the generated-vs-reference check")
	}

	assertImportedDeepParityCases(t, "tsx", []struct {
		name string
		src  string
	}{
		{
			name: "unary_negation_over_call_after_function_type_declaration",
			src:  "let NodeConstructor: (kind: number) => Node;\n!isNodeKind(kind);\n",
		},
		{
			name: "logical_or_over_calls_after_function_type_declaration",
			src:  "let NodeConstructor: (kind: number) => Node;\na() || b();\n",
		},
	})
}

// TestTypeScriptConsecutiveDecoratedFieldsKnownIssue documents a second,
// DISTINCT gotreesitter defect found in the same investigation: 2 of 3
// consecutive `@decorator field: T = value;` class members can be lost
// (collapsed into a stray `decorator` node ahead of the next member,
// wrapped in an ERROR) when the class body's `identifier ":"` shape has its
// own locally-unresolvable declared conflict (_property_name vs. a shift
// that continues toward `type_annotation`, with NO precedence signal on
// either side — unlike the unary/call case above) AND that conflict's GLR
// fork is still alive when a second field is reached.
//
// Root cause, precisely: the fork's 2 stack lineages reconverge on the
// identical (state, byte-offset) pair but never actually merge, because the
// pre-existing ("orig") stack stays on the fast entries-only representation
// (see glrStack.entries) while its sibling fork is GSS-backed — the mixed
// entries-vs-GSS equivalence path (stackEquivalentForLanguageWithScratch)
// then does a structural deep-frontier comparison that (correctly) sees a
// different node at one frame (a hidden _property_name wrapper vs. the bare
// identifier it wraps) and refuses to collapse the two lineages, even though
// they are behaviorally interchangeable from that point on. This is a
// generic GLR merge/result-selection gap, not a TypeScript grammar-table
// defect — real tree-sitter's C reference merges purely on (state,
// byte-offset) ("header-only" equivalence, see stacksHeaderEquivalent's own
// doc comment), a cheaper and more permissive criterion than gotreesitter's
// current deliberately-conservative deep-frontier check. Fixing it safely
// requires a cross-language change to glr.go's stack-equivalence/result-
// selection machinery (shared by 200+ grammars) with much broader
// verification than a TypeScript-scoped change can responsibly cover — see
// the session trace for the full investigation.
//
// Reproduction needs GOT_GLR_MAX_STACKS raised above the default cap (the
// Docker real-corpus harness runs TypeScript with GOT_GLR_MAX_STACKS=64) and
// a class body with at least 2 members, so it is environment-sensitive and
// currently skipped rather than asserted. Un-skip once the GSS
// entries-vs-GSS merge gap above is fixed.
func TestTypeScriptConsecutiveDecoratedFieldsKnownIssue(t *testing.T) {
	t.Skip("known issue: generic GLR entries-vs-GSS merge gap (see doc comment); needs GOT_GLR_MAX_STACKS raised above default to reproduce, not yet fixed")

	if raceEnabled {
		t.Skip("skip heavyweight TypeScript parity generation under -race; non-race coverage keeps the generated-vs-reference check")
	}

	assertImportedDeepParityCases(t, "typescript", []struct {
		name string
		src  string
	}{
		{
			name: "three_consecutive_decorated_class_fields",
			src: "@baz @bam class Foo {\n" +
				"    @foo static 2: string;\n" +
				"    @bar.buzz(grue) public static 2: string = 'string';\n" +
				"    @readonly readonly 'hello'?: int = 'string';\n" +
				"    @readonly fooBar(@required param: any, @optional param2?: any) {\n" +
				"    }\n" +
				"}\n",
		},
	})
}
