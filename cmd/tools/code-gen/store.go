package main
//
//import (
//	"fmt"
//	. "github.com/dave/jennifer/jen"
//	"github.com/pkg/errors"
//	"go/ast"
//	"go/types"
//	"golang.org/x/tools/go/packages"
//	"strings"
//)
//
//type StructType struct {
//	Name string
//	Plural string
//	st *types.Struct
//	Fields []Field
//}
//
//type Field struct {
//	Tags []string
//	Field *types.Var
//}
//
//func generateStore(sourceType string) (string, error) {
//	structType, name, err := loadStruct(sourceType)
//	if err != nil {
//		return "", err
//	}
//
//	fields := []Field{}
//
//	for i := 0; i < structType.NumFields(); i++ {
//		//ast.
//		field := structType.Field(i)
//
//		f := Field{Field: field, Tags: []string{}}
//
//		tag := structType.Tag(0)
//		for i := 1; tag != ""; i++ {
//			f.Tags = append(f.Tags, tag)
//			tag = structType.Tag(i)
//		}
//
//		fields = append(fields, f)
//		//fmt.Println(field.Name(), tagValue, field.Type())
//	}
//
//	st := StructType{
//		Name:   name,
//		Plural: "",
//		st:     structType,
//		Fields: fields,
//	}
//
//}
//
//func generateCreate() {
//
//}
//
//
//
//
//func loadStruct(sourceType string) (*types.Struct, string, error) {
//	sourceTypePackage, sourceTypeName, err := splitSourceType(sourceType)
//	if err != nil {
//		return nil, "", errors.Wrap(err, "failed to split source type")
//	}
//
//	pkg, err := loadPackage(sourceTypePackage)
//	if err != nil {
//		return nil, "", errors.Wrap(err, "failed to load package")
//	}
//
//	// 3. Lookup the given source type name in the package declarations
//	obj := pkg.Types.Scope().Lookup(sourceTypeName)
//	if obj == nil {
//		return nil, "", fmt.Errorf("%s not found in declared types of %s",
//			sourceTypeName, pkg)
//	}
//
//	// 4. We check if it is a declared type
//	if _, ok := obj.(*types.TypeName); !ok {
//		return nil, "", fmt.Errorf("%v is not a named type", obj)
//	}
//	// 5. We expect the underlying type to be a struct
//	structType, ok := obj.Type().Underlying().(*types.Struct)
//	if !ok {
//		return nil, "", fmt.Errorf("type %v is not a struct", obj)
//	}
//
//	return structType, sourceTypeName, nil
//}
//
//func splitSourceType(sourceType string) (string, string, error) {
//	idx := strings.LastIndexByte(sourceType, '.')
//	if idx == -1 {
//		return "", "", fmt.Errorf(`expected qualified type as "pkg/path.MyType"`)
//	}
//	sourceTypePackage := sourceType[0:idx]
//	sourceTypeName := sourceType[idx+1:]
//	return sourceTypePackage, sourceTypeName, nil
//}
//
//func loadPackage(path string) (packages.Package, error) {
//	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedImports}
//	pkgs, err := packages.Load(cfg, path)
//	if err != nil {
//		return packages.Package{}, fmt.Errorf("loading packages for inspection: %v", err)
//	}
//	if packages.PrintErrors(pkgs) > 0 {
//		return packages.Package{}, fmt.Errorf("encounter packages errors: %s", pkgs[0].Errors)
//	}
//
//	return *pkgs[0], nil
//}
