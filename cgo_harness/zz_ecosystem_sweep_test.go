//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TestEcosystemSweep runs the ECOSYSTEM structural gate across every grammar
// that has a C oracle (languages.lock) and a corpus subdir under REPRO_DIR.
// Per file: bucket recovery (both error) / goFail (only Go errors on
// C-clean input) / structural-clean / structural-diverge (compareNodes on
// clean-both files). Run with GTS_PARITY_COMPARE_SPANS=0 GTS_PARITY_COMPARE_FIELDS=1.
//
// One line per grammar:
//
//	ECO-SWEEP <name> clean=<m>/<cleanFiles> diverge=<d> goFail=<g> recovery=<r> files=<n>
//
// A grammar is ecosystem-clean when diverge==0 && goFail==0.
func TestEcosystemSweep(t *testing.T) {
	root := os.Getenv("REPRO_DIR")
	if root == "" {
		t.Skip("set REPRO_DIR")
	}
	n := 40
	if v := os.Getenv("REPRO_N"); v != "" {
		fmt.Sscanf(v, "%d", &n)
	}

	type row struct{ clean, cleanFiles, diverge, goFail, recovery, files int }
	rows := map[string]row{}
	var order []string

	for _, e := range grammars.AllLanguages() {
		name := e.Name
		dir := filepath.Join(root, name)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}
		cLang, err := ParityCLanguage(name)
		if err != nil {
			continue
		}
		goLang := e.Language()
		if goLang == nil || len(e.Extensions) == 0 {
			continue
		}

		var files []string
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || strings.Contains(p, "/.git/") {
				return nil
			}
			if info.Size() < 32 || info.Size() > 200_000 {
				return nil
			}
			base := strings.ToLower(filepath.Base(p))
			for _, ext := range e.Extensions {
				ext = strings.ToLower(strings.TrimSpace(ext))
				if ext != "" && strings.HasSuffix(base, ext) {
					files = append(files, p)
					break
				}
			}
			return nil
		})
		sort.Strings(files)
		if n < len(files) {
			files = files[:n]
		}
		if len(files) == 0 {
			continue
		}

		var r row
		r.files = len(files)
		for _, f := range files {
			func() {
				defer func() { _ = recover() }() // a bad file must not kill the sweep
				src, rerr := os.ReadFile(f)
				if rerr != nil || len(src) == 0 {
					return
				}
				gp := gts.NewParser(goLang)
				gtree, _ := gp.Parse(src)
				if gtree == nil || gtree.RootNode() == nil {
					return
				}
				defer gtree.Release()
				goErr := gtree.RootNode().HasError()
				cp := sitter.NewParser()
				defer cp.Close()
				_ = cp.SetLanguage(cLang)
				ctree := cp.Parse(src, nil)
				if ctree == nil || ctree.RootNode() == nil {
					return
				}
				defer ctree.Close()
				cErr := ctree.RootNode().HasError()
				if goErr || cErr {
					if goErr && !cErr {
						r.goFail++
					} else {
						r.recovery++
					}
					return
				}
				r.cleanFiles++
				var errs []string
				compareNodes(gtree.RootNode(), goLang, ctree.RootNode(), "root", &errs)
				if len(errs) == 0 {
					r.clean++
				} else {
					r.diverge++
				}
			}()
		}
		rows[name] = r
		order = append(order, name)
		fmt.Printf("ECO-SWEEP %s clean=%d/%d diverge=%d goFail=%d recovery=%d files=%d\n",
			name, r.clean, r.cleanFiles, r.diverge, r.goFail, r.recovery, r.files)
	}

	ecoClean, hasDiverge, hasGoFail := 0, 0, 0
	for _, name := range order {
		r := rows[name]
		if r.diverge == 0 && r.goFail == 0 {
			ecoClean++
		}
		if r.diverge > 0 {
			hasDiverge++
		}
		if r.goFail > 0 {
			hasGoFail++
		}
	}
	fmt.Printf("ECO-SWEEP-SUMMARY grammars=%d ecosystemClean=%d hasStructDiverge=%d hasGoFail=%d\n",
		len(order), ecoClean, hasDiverge, hasGoFail)
}
