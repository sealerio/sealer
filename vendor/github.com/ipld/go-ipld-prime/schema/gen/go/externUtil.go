package gengo

import (
	"go/ast"
	"go/parser"
	"go/token"
	path "path/filepath"
	"strings"
)

// getExternTypes provides a mapping of all types defined in the destination package.
// It is used by generate to not duplicate defined types to allow overriding of types.
func getExternTypes(pth string) (map[string]struct{}, error) {
	set := token.NewFileSet()
	packs, err := parser.ParseDir(set, pth, nil, 0)
	if err != nil {
		return nil, err
	}

	types := make(map[string]struct{})
	for _, pack := range packs {
		for fname, f := range pack.Files {
			if strings.HasPrefix(path.Base(fname), "ipldsch_") {
				continue
			}
			for _, d := range f.Decls {
				if t, isType := d.(*ast.GenDecl); isType {
					if t.Tok == token.TYPE {
						for _, s := range t.Specs {
							ts := s.(*ast.TypeSpec)
							types[ts.Name.Name] = struct{}{}
						}
					}
				}
			}
		}
	}

	return types, nil
}
