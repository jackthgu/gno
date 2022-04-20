package gno

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"go.uber.org/multierr"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	gnoRealmPkgsPrefixBefore = "gno.land/r/"
	gnoRealmPkgsPrefixAfter  = "github.com/gnolang/gno/examples/gno.land/r/"
	gnoPackagePrefixBefore   = "gno.land/p/"
	gnoPackagePrefixAfter    = "github.com/gnolang/gno/examples/gno.land/p/"
	gnoStdPkgBefore          = "std"
	gnoStdPkgAfter           = "github.com/gnolang/gno/stdlibs/stdshim"
)

var stdlibWhitelist = []string{
	"regexp",
	"std",
	"strconv",
	"strings",
	"maths",
}

func Precompile(fset *token.FileSet, f *ast.File) (string, error) {
	var out bytes.Buffer

	transformed, err := precompileAST(fset, f)
	if err != nil {
		return "", fmt.Errorf("precompile ast: %w", err)
	}

	_, err = out.WriteString("// Code generated by github.com/gnolang/gno. DO NOT EDIT.\n\n//go:build gno\n// +build gno\n\n")
	if err != nil {
		return "", fmt.Errorf("write to buffer: %w", err)
	}
	err = format.Node(&out, fset, transformed)
	return out.String(), nil
}
func precompileAST(fset *token.FileSet, f *ast.File) (ast.Node, error) {
	var errs error

	imports := astutil.Imports(fset, f)

	// import whitelist
	for _, paragraph := range imports {
		for _, importSpec := range paragraph {
			importPath := strings.TrimPrefix(strings.TrimSuffix(importSpec.Path.Value, `"`), `"`)

			if strings.HasPrefix(importPath, gnoRealmPkgsPrefixBefore) {
				continue
			}

			if strings.HasPrefix(importPath, gnoPackagePrefixBefore) {
				continue
			}

			valid := false
			for _, whitelisted := range stdlibWhitelist {
				if importPath == whitelisted {
					valid = true
					continue
				}
			}
			if valid {
				continue
			}

			errs = multierr.Append(errs, fmt.Errorf("import %q is not in the whitelist", importPath))
		}
	}

	// rewrite imports
	for _, paragraph := range imports {
		for _, importSpec := range paragraph {
			importPath := strings.TrimPrefix(strings.TrimSuffix(importSpec.Path.Value, `"`), `"`)

			// std package
			if importPath == gnoStdPkgBefore {
				if !astutil.RewriteImport(fset, f, gnoStdPkgBefore, gnoStdPkgAfter) {
					errs = multierr.Append(errs, fmt.Errorf("failed to replace the %q package with %q", gnoStdPkgBefore, gnoStdPkgAfter))
				}
			}

			// p/pkg packages
			if strings.HasPrefix(importPath, gnoPackagePrefixBefore) {
				target := gnoPackagePrefixAfter + strings.TrimPrefix(importPath, gnoPackagePrefixBefore)

				if !astutil.RewriteImport(fset, f, importPath, target) {
					errs = multierr.Append(errs, fmt.Errorf("failed to replace the %q package with %q", importPath, target))
				}

			}

			// r/realm packages
			if strings.HasPrefix(importPath, gnoRealmPkgsPrefixBefore) {
				target := gnoRealmPkgsPrefixAfter + strings.TrimPrefix(importPath, gnoRealmPkgsPrefixBefore)

				if !astutil.RewriteImport(fset, f, importPath, target) {
					errs = multierr.Append(errs, fmt.Errorf("failed to replace the %q package with %q", importPath, target))
				}

			}
		}
	}

	// custom handler
	node := astutil.Apply(f,
		// pre
		func(c *astutil.Cursor) bool {
			// do things here
			return true
		},
		// post
		func(c *astutil.Cursor) bool {
			// and here
			return true
		},
	)

	return node, errs
}
