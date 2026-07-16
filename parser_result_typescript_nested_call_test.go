package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestTypeScriptNestedGenericCall covers the TypeScript CALL-context nested-
// generic bug: `callee<Outer<Inner>>(args)` in an expression position is
// mis-parsed (only under accumulated GLR context) as a `<`/`>` comparison chain
// with a residual ERROR instead of a call_expression(function, type_arguments,
// arguments). The reconstruction re-parses the offending span in isolation —
// where it parses correctly — and splices the clean call back.
//
// The reproducers embed the calls in an if/else + async-function context
// because the isolated one-liner parses correctly; the bug needs the
// surrounding state (mirrors go-webauthn WebauthnService.ts).
func TestTypeScriptNestedGenericCall(t *testing.T) {
	lang := grammars.TypescriptLanguage()

	cases := []struct {
		name string
		src  string
	}{
		{
			"await_member_call_if_else",
			"async function g(d: boolean): Promise<X> {\n" +
				"  let response: AxiosResponse<ServiceResponse<CredentialCreation>>;\n" +
				"  if (d) {\n" +
				"    response = await axios.get<ServiceResponse<CredentialCreation>>(P1);\n" +
				"  } else {\n" +
				"    response = await axios.get<ServiceResponse<CredentialCreation>>(P2);\n" +
				"  }\n" +
				"  return response;\n" +
				"}\n",
		},
		{
			"return_member_call_if_else",
			"async function h(c: boolean): Promise<X> {\n" +
				"  if (c) {\n" +
				"    return axios.post<ServiceResponse<SignInResponse>>(P1, j);\n" +
				"  }\n" +
				"  return axios.post<OptionalDataServiceResponse<any>>(P2, j);\n" +
				"}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := gts.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			sx := root.SExpr(lang)
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", sx)
			}
			// Each mis-parsed call must recover to a call_expression carrying a
			// type_arguments field (the nested generic), not a comparison chain.
			call := findTSCallWithTypeArguments(root, lang)
			if call == nil {
				t.Fatalf("no call_expression with type_arguments produced\n%s", sx)
			}
			if strings.Contains(sx, "(ERROR") {
				t.Fatalf("residual ERROR node remains\n%s", sx)
			}
		})
	}

	// The curried generic-call declaration (zustand store idiom) collapses the
	// whole declaration into a statement-level ERROR; it must recover to
	// export_statement > lexical_declaration > variable_declarator whose value is
	// a curried call_expression(call_expression(create, type_arguments, args()),
	// args(curry)). The trailing comma inside the curry must survive.
	curried := []struct {
		name string
		src  string
	}{
		{"exported_curried_store", "export const useSession = create<SessionState>()(persist(cfg));\n"},
		{"exported_curried_trailing_comma", "export const useSession = create<SessionState>()(persist(cfg),);\n"},
		{"local_curried_store", "const s = build<State>()(mw(fn));\n"},
	}
	for _, tc := range curried {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := gts.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			sx := root.SExpr(lang)
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", sx)
			}
			// The inner generic call must recover its type_arguments, and the
			// declaration must be a proper lexical_declaration (not an ERROR).
			if findTSCallWithTypeArguments(root, lang) == nil {
				t.Fatalf("no inner call_expression with type_arguments\n%s", sx)
			}
			if strings.Contains(sx, "(ERROR") {
				t.Fatalf("residual ERROR node remains\n%s", sx)
			}
			if findTSNodeByType(root, lang, "lexical_declaration") == nil {
				t.Fatalf("no lexical_declaration produced\n%s", sx)
			}
		})
	}

	// Genuine comparisons must never be rewritten into a generic call: the
	// reconstruction must not invent a type_arguments field. The second case
	// deliberately trips an unrelated pre-existing `>>`-in-arguments parse
	// error; the reconstruction must still leave it alone (fire ONLY when the
	// isolated re-parse yields a clean call_expression), so we assert no
	// spurious type_arguments call rather than absence of the error.
	negatives := []struct {
		name string
		src  string
	}{
		{"clean_comparison", "const ok = count < items.length;\n"},
		{"errored_shift_in_args", "const r = f(a < b, c >> d);\n"},
	}
	for _, tc := range negatives {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := gts.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if findTSCallWithTypeArguments(root, lang) != nil {
				t.Fatalf("genuine comparison wrongly grew a type_arguments call\n%s", root.SExpr(lang))
			}
		})
	}
}

// findTSNodeByType returns the first node of the given type in preorder.
func findTSNodeByType(n *gts.Node, lang *gts.Language, typ string) *gts.Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == typ {
		return n
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if got := findTSNodeByType(n.Child(i), lang, typ); got != nil {
			return got
		}
	}
	return nil
}

// findTSCallWithTypeArguments returns the first call_expression that has a
// type_arguments field child.
func findTSCallWithTypeArguments(n *gts.Node, lang *gts.Language) *gts.Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == "call_expression" && n.ChildByFieldName("type_arguments", lang) != nil {
		return n
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if got := findTSCallWithTypeArguments(n.Child(i), lang); got != nil {
			return got
		}
	}
	return nil
}
