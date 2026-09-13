//go:build gts_parsercorephase0

package gotreesitter_test

import (
	gotreesitter "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

func init() {
	gotreesitter.SetDiagnosticParserCoreWarmGoScannerForTest(grammars.GoExternalScanner{})
}
