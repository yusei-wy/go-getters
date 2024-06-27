package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Getters struct {
	PackageName string
	Imports     []string
	Fields      []GetterField
}

type GetterField struct {
	StructName string
	MethodName string
	FieldName  string
	FieldType  string
}

func NewGetterField(structName, methodName, filedName, filedType string) GetterField {
	return GetterField{structName, methodName, filedName, filedType}
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fileNames, err := recursiveSeachGoFileNames(dir)
	if err != nil {
		panic(err)
	}

	for _, fileName := range fileNames {
		if !strings.HasSuffix(fileName, ".go") {
			continue
		}

		targetStructs, packageName, imports, err := seachTargetStructs(fileName)
		if err != nil {
			panic(err)
		}

		if len(targetStructs) > 0 {
			getters := createGetters(packageName, imports, targetStructs)
			generateGetters(fileName, getters)
		}
	}
}

func recursiveSeachGoFileNames(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func seachTargetStructs(fileName string) ([]*ast.TypeSpec, string, []string, error) {
	packageName := ""
	imports := []string{}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		return nil, packageName, imports, err
	}

	var structs []*ast.TypeSpec
	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		if genDecl.Tok != token.TYPE || genDecl.Doc == nil {
			return true
		}

		packageName = node.Name.Name
		imports = make([]string, len(node.Imports))
		for i, importSpec := range node.Imports {
			imports[i] = importSpec.Path.Value[1 : len(importSpec.Path.Value)-1]
			if err != nil {
				return true
			}
		}

		for _, comment := range genDecl.Doc.List {
			if strings.HasPrefix(comment.Text, "//go:generate getters") {
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						structs = append(structs, typeSpec)
					}
				}
			}
		}

		return true
	})

	return structs, packageName, imports, nil
}

type UsedImport struct {
	Name string
	Used bool
}

func createGetters(packageName string, imports []string, typeSpecs []*ast.TypeSpec) *Getters {
	getters := Getters{PackageName: packageName}

	usedImportMap := make(map[string]UsedImport)
	for _, name := range imports {
		paths := strings.Split(name, "/")
		shortName := paths[len(paths)-1]
		usedImportMap[shortName] = UsedImport{
			Name: name,
			Used: false,
		}
	}

	for _, typeSpec := range typeSpecs {
		structName := typeSpec.Name.Name
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		getterFields := make([]GetterField, len(structType.Fields.List))
		for i, field := range structType.Fields.List {
			filedName := field.Names[0].Name
			methodName := cases.Title(language.Und).String(filedName)
			fieldType := getFiledTypeString(field.Type)

			getterFields[i] = NewGetterField(structName, methodName, filedName, fieldType)

			shortFiledType := strings.Split(fieldType, ".")[0]
			if imp, ok := usedImportMap[shortFiledType]; ok {
				imp.Used = true
				usedImportMap[shortFiledType] = imp
			}
		}

		getters.Fields = append(getters.Fields, getterFields...)
	}

	userdImports := make([]string, 0, len(usedImportMap))
	for _, imp := range usedImportMap {
		if imp.Used {
			userdImports = append(userdImports, imp.Name)
		}
	}

	getters.Imports = userdImports

	return &getters
}

func getFiledTypeString(expr ast.Expr) string {
	// NOTE: Unsupport struct type and func type
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return "*" + getFiledTypeString(expr.X)
	case *ast.SelectorExpr:
		return getFiledTypeString(expr.X) + "." + expr.Sel.Name
	case *ast.ArrayType:
		return "[]" + getFiledTypeString(expr.Elt)
	case *ast.MapType:
		return "map[" + getFiledTypeString(expr.Key) + "]" + getFiledTypeString(expr.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chann " + getFiledTypeString(expr.Value)
	case *ast.Ellipsis:
		return "..." + getFiledTypeString(expr.Elt)
	default:
		panic(fmt.Sprintf("unsupported type: %T", expr))
	}
}

func generateGetters(fileName string, getters *Getters) {
	tmpl, err := template.ParseFiles("getters.tmpl")
	if err != nil {
		panic(err)
	}

	outputFileName := strings.TrimSuffix(fileName, ".go") + "_getters.go"
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, getters)
	if err != nil {
		panic(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(outputFileName, formatted, 0o664)
	if err != nil {
		panic(err)
	}

	fmt.Printf("generated getters for struct: %s\n", fileName)
}
