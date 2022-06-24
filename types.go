package swag

import (
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/go-openapi/spec"
)

// Schema parsed schema.
type Schema struct {
	*spec.Schema        //
	PkgPath      string // package import path used to rename Name of a definition int case of conflict
	Name         string // Name in definitions
}

// TypeSpecDef the whole information of a typeSpec.
type TypeSpecDef struct {
	// ast file where TypeSpec is
	File *ast.File

	// the TypeSpec of this type definition
	TypeSpec *ast.TypeSpec

	// path of package starting from under ${GOPATH}/src or from module path in go.mod
	PkgPath    string
	ParentSpec ast.Decl
}

// Name the name of the typeSpec.
func (t *TypeSpecDef) Name() string {
	if t.TypeSpec != nil {
		return t.TypeSpec.Name.Name
	}

	return ""
}

// FullName full name of the typeSpec.
func (t *TypeSpecDef) FullName() string {
	var fullName string
	if parentFun, ok := (t.ParentSpec).(*ast.FuncDecl); ok && parentFun != nil {
		fullName = fullTypeNameFunctionScoped(t.File.Name.Name, parentFun.Name.Name, t.TypeSpec.Name.Name)
	} else {
		fullName = fullTypeName(t.File.Name.Name, t.TypeSpec.Name.Name)
	}
	return fullName
}

// FullPath of the typeSpec.
func (t *TypeSpecDef) FullPath() string {
	return t.PkgPath + "." + t.Name()
}

// AstFileInfo information of an ast.File.
type AstFileInfo struct {
	// File ast.File
	File *ast.File

	// Path the path of the ast.File
	Path string

	// PackagePath package import path of the ast.File
	PackagePath string
}

// PackageDefinitions files and definition in a package.
type PackageDefinitions struct {
	// files in this package, map key is file's relative path starting package path
	files     []*ast.File
	filenames []string

	// definitions in this package, map key is typeName
	TypeDefinitions map[string]*TypeSpecDef

	// package name
	ImportName string
	ImportPath string

	// package dir
	Dir string
}

func newPackageDefinitions(importName, importPath, packageDir string) (*PackageDefinitions, error) {
	fis, err := ioutil.ReadDir(packageDir)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, fi := range fis {
		if ext := filepath.Ext(fi.Name()); strings.ToLower(ext) != ".go" {
			continue
		}
		if strings.HasSuffix(fi.Name(), "_test.go") {
			continue
		}
		if strings.HasSuffix(fi.Name(), "-gen.go") {
			continue
		}

		filenames = append(filenames, fi.Name())
	}

	packageDir = strings.TrimSuffix(packageDir, "/")
	packageDir = strings.TrimSuffix(packageDir, "\\")

	log.Println("load package -", packageDir, filenames)

	return &PackageDefinitions{
		// files in this package, map key is file's relative path starting package path
		files:           make([]*ast.File, len(filenames)),
		filenames:       filenames,
		TypeDefinitions: make(map[string]*TypeSpecDef),
		ImportName: importName,
		ImportPath: importPath,
		Dir:        packageDir,
	}, nil
}

func (pd *PackageDefinitions) mustAdd(filename string, file *ast.File) {
	simplefilename := strings.TrimLeft(strings.TrimPrefix(filename, pd.Dir), "/\\")

	for idx, name := range pd.filenames {
		if simplefilename == name {
			pd.files[idx] = file
			return
		}
	}

	println("**********", pd.Dir)
	for _, name := range pd.filenames {
		println(name)
	}
	panic(errors.New(filename + " isnot exist in the " + pd.Dir))
}

func (pd *PackageDefinitions) findFile(filename string) *ast.File {
	simplefilename := strings.TrimLeft(strings.TrimPrefix(filename, pd.Dir), "/\\")

	for idx, name := range pd.filenames {
		if simplefilename == name {
			return pd.files[idx]
		}
	}
	return nil
}

func (pd *PackageDefinitions) fileCount() int {
	return len(pd.filenames)
}

func (pd *PackageDefinitions) loadFileByIndex(idx int) (*ast.File, bool, error) {
	if pd.files[idx] != nil {
		return pd.files[idx], false, nil
	}

	filename := filepath.Join(pd.Dir, pd.filenames[idx])

	fileTree, err := goparser.ParseFile(token.NewFileSet(), filename, nil, goparser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("cannot parse source files %s: %s", filename, err)
	}

	pd.files[idx] = fileTree
	return fileTree, true, nil
}
