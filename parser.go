package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"io/ioutil"
	"strings"
)

func ClassesFromDir(dir string) []Class {
	classes := []Class{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "spec.go") {
			continue
		}
		filename := dir + "/" + file.Name()
		class := classFromFile(filename)
		if class.Line != 0 {
			class.Filename = strings.Replace(class.Name, ".go", "", 1)
			classes = append(classes, class)
		}
	}
	return classes
}

func classFromFile(filepath string) Class {
	allMethods := []Method{}
	class := Class{}
	class.SetClassname(filepath)

	// Parse target file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return class
	}

	// Get comments
	allComments := AllComments{fset, f.Comments}
	// ast.Print(fset, f.Comments)

	// Find class & methods
	var methods *ast.ValueSpec
	// Loop through declarations
	for _, decl := range f.Decls {
		// Continue only for general declarations
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				// Assign class line number if found
				if tSpec, ok := spec.(*ast.TypeSpec); ok && class.MatchName(tSpec.Name.Name) {
					node := tSpec.Name
					class.Line = fset.Position(node.NamePos).Line
				}
				// Assign methods if found
				if vSpec, ok := spec.(*ast.ValueSpec); ok && class.MatchInstanceMethods(vSpec.Names[0].Name) {
					methods = vSpec
				}
			}
		}
	}

	// Return blank class if class definition is not found
	if class.Line == 0 {
		return class
	}
	// Retrieve class comments
	comments := allComments.findCommentFor(class.Line)
	class.Comment = template.HTML(comments.Description)

	// Return class if there is not built-in methods
	if methods == nil {
		return class
	}

	// Loop through methods to find each method
	allExpr := methods.Values[0].(*ast.CompositeLit).Elts
	var attrs []ast.Expr
	for _, expr := range allExpr {
		attrs = expr.(*ast.CompositeLit).Elts
		method := Method{}
		// Attributes should only contain "Name" & "Fn" for now
		for _, attr := range attrs {
			thisExpr := attr.(*ast.KeyValueExpr)
			name := thisExpr.Key.(*ast.Ident).Name
			if name == "Name" {
				method.FnName = strings.Replace(thisExpr.Value.(*ast.BasicLit).Value, "\"", "", -1)
				method.FnLine = fset.Position(thisExpr.Key.(*ast.Ident).NamePos).Line
			}
			if name == "Fn" {
				methodComments := allComments.findCommentFor(method.FnLine)
				method.Params = methodComments.Params
				method.Returns = methodComments.Returns
				method.Comment = template.HTML(methodComments.Description)
			}
		}
		allMethods = append(allMethods, method)
	}

	class.InstanceMethods = allMethods
	return class
}

func Write(filepath string, classes []Class) {
	b, err := json.Marshal(classes)
	if err != nil {
		panic(err)
	}
	fmt.Println("Generated:", filepath)
	err = ioutil.WriteFile(filepath, b, 0644)
	if err != nil {
		panic(err)
	}
}

func InsertClassLinks(classes Classes) Classes {
	var returned_classes Classes
	puncs := []string{" ", ",", ".", ";", "\n"}
	// loop classes
	for _, class := range classes {
		text := string(class.Comment)
		// insert link to class comment
		for _, each_class := range classes {
			each_class_link := " [" + each_class.Name + "](" + each_class.Filename + ".html) "
			for _, sym := range puncs {
				text = strings.Replace(text, " "+each_class.Name+sym, each_class_link, -1)
			}
		}

		// loop methods in a class
		for i, method := range class.InstanceMethods {
			text := string(method.Comment)
			// insert link to method comment
			for _, each_class := range classes {
				each_class_link := " [" + each_class.Name + "](" + each_class.Filename + ".html) "
				for _, sym := range puncs {
					text = strings.Replace(text, " "+each_class.Name+sym, each_class_link, -1)
				}
			}
			class.InstanceMethods[i].Comment = template.HTML(text)
		}

		class.Comment = template.HTML(text)
		returned_classes = append(returned_classes, class)
	}

	return returned_classes
}
