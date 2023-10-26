package typegen

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

var (
	knownPackageNamesMu sync.Mutex
	pkgNameToPkgPath    = make(map[string]string)
	pkgPathToPkgName    = make(map[string]string)

	defaultImports = []Import{
		{Name: "cbg", PkgPath: "github.com/whyrusleeping/cbor-gen"},
		{Name: "xerrors", PkgPath: "golang.org/x/xerrors"},
		{Name: "cid", PkgPath: "github.com/ipfs/go-cid"},
	}
)

func init() {
	for _, imp := range defaultImports {
		if was, conflict := pkgNameToPkgPath[imp.Name]; conflict {
			panic(fmt.Sprintf("reused pkg name %s for %s and %s", imp.Name, imp.PkgPath, was))
		}
		if _, conflict := pkgPathToPkgName[imp.Name]; conflict {
			panic(fmt.Sprintf("duplicate default import %s", imp.PkgPath))
		}
		pkgNameToPkgPath[imp.Name] = imp.PkgPath
		pkgPathToPkgName[imp.PkgPath] = imp.Name
	}
}

func resolvePkgName(path, typeName string) string {
	parts := strings.Split(typeName, ".")
	if len(parts) != 2 {
		panic(fmt.Sprintf("expected type to have a package name: %s", typeName))
	}
	defaultName := parts[0]

	knownPackageNamesMu.Lock()
	defer knownPackageNamesMu.Unlock()

	// Check for a known name and use it.
	if name, ok := pkgPathToPkgName[path]; ok {
		return name
	}

	// Allocate a name.
	for i := 0; ; i++ {
		tryName := defaultName
		if i > 0 {
			tryName = fmt.Sprintf("%s%d", defaultName, i)
		}
		if _, taken := pkgNameToPkgPath[tryName]; !taken {
			pkgNameToPkgPath[tryName] = path
			pkgPathToPkgName[path] = tryName
			return tryName
		}
	}

}

type Import struct {
	Name, PkgPath string
}

func ImportsForType(currPkg string, t reflect.Type) []Import {
	switch t.Kind() {
	case reflect.Array, reflect.Slice, reflect.Ptr:
		return ImportsForType(currPkg, t.Elem())
	case reflect.Map:
		return dedupImports(append(ImportsForType(currPkg, t.Key()), ImportsForType(currPkg, t.Elem())...))
	default:
		path := t.PkgPath()
		if path == "" || path == currPkg {
			// built-in or in current package.
			return nil
		}

		return []Import{{PkgPath: path, Name: resolvePkgName(path, t.String())}}
	}
}

func dedupImports(imps []Import) []Import {
	impSet := make(map[string]string, len(imps))
	for _, imp := range imps {
		impSet[imp.PkgPath] = imp.Name
	}
	deduped := make([]Import, 0, len(imps))
	for pkg, name := range impSet {
		deduped = append(deduped, Import{Name: name, PkgPath: pkg})
	}
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].PkgPath < deduped[j].PkgPath
	})
	return deduped
}
