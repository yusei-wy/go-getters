package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed *.tmpl
var tmplFiles embed.FS

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

	fileNames, err := recursiveSearchGoFileNames(dir)
	if err != nil {
		panic(err)
	}

	for _, fileName := range fileNames {
		if !strings.HasSuffix(fileName, ".go") {
			continue
		}

		targetStructs, packageName, imports, err := searchTargetStructs(fileName)
		if err != nil {
			panic(err)
		}

		if len(targetStructs) > 0 {
			getters := createGetters(packageName, imports, targetStructs)
			generateGetters(fileName, getters)
		}
	}
}

func recursiveSearchGoFileNames(dir string) ([]string, error) {
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

func searchTargetStructs(fileName string) ([]*ast.TypeSpec, string, []string, error) {
	packageName := ""
	imports := []string{}

	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, fileName, nil, parser.ParseComments)
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
			methodName := strings.ToUpper(filedName[0:1]) + filedName[1:]
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

	usedImports := make([]string, 0, len(usedImportMap))
	for _, imp := range usedImportMap {
		if imp.Used {
			usedImports = append(usedImports, imp.Name)
		}
	}

	getters.Imports = usedImports

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
	tmpl, err := template.ParseFS(tmplFiles, "getters.tmpl")
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
