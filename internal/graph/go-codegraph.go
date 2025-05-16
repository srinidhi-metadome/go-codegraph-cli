package graph

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProjectStructure represents the entire project structure
type ProjectStructure struct {
	Project   map[string]PackageInfo `json:"project"`
	CodeGraph CodeGraph              `json:"codeGraph"`
}

// PackageInfo represents information about a Go package
type PackageInfo struct {
	Modules map[string]ModuleInfo `json:"modules"`
}

// ModuleInfo represents information about a Go file
type ModuleInfo struct {
	Structs      []StructInfo    `json:"structs"`
	Functions    []FunctionInfo  `json:"functions"`
	Interfaces   []InterfaceInfo `json:"interfaces"`
	Dependencies []string        `json:"dependencies"`
	Constants    []ConstantInfo  `json:"constants"`
	Variables    []VariableInfo  `json:"variables"`
}

// StructInfo represents information about a Go struct
type StructInfo struct {
	Name       string         `json:"name"`
	Functions  []FunctionInfo `json:"functions"` // Methods
	Properties []PropertyInfo `json:"properties"`
	Comment    string         `json:"comment,omitempty"`
	ID         string         `json:"id"`
}

// InterfaceInfo represents information about a Go interface
type InterfaceInfo struct {
	Name      string         `json:"name"`
	Functions []FunctionInfo `json:"functions"`
	Comment   string         `json:"comment,omitempty"`
	ID        string         `json:"id"`
}

// FunctionInfo represents information about a Go function
type FunctionInfo struct {
	Name       string          `json:"name"`
	Parameters []ParameterInfo `json:"parameters"`
	ReturnType string          `json:"returnType"`
	Content    string          `json:"content,omitempty"`
	Comment    string          `json:"comment,omitempty"`
	ID         string          `json:"id"`
	Package    string          `json:"package,omitempty"`
	FilePath   string          `json:"filePath,omitempty"`
}

// PropertyInfo represents information about a struct field
type PropertyInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Comment string `json:"comment,omitempty"`
}

// ParameterInfo represents information about a function parameter
type ParameterInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ConstantInfo represents information about a constant
type ConstantInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id"`
}

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
	ID    string `json:"id"`
}

// CodeGraph represents the relationships between code entities
type CodeGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node represents a single entity in the code graph
type Node struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Package string `json:"package,omitempty"`
	File    string `json:"file,omitempty"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// Global variable to store nodes and edges
var (
	nodes     = make(map[string]Node)
	edges     = []Edge{}
	funcMap   = make(map[string]string) // Maps function name to ID
	structMap = make(map[string]string) // Maps struct name to ID
	typeMap   = make(map[string]string) // Maps type name to ID
	idCounter = 0
)

// Helper function to generate unique IDs
func generateID(prefix string) string {
	idCounter++
	return prefix + strconv.Itoa(idCounter)
}

func extractComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	var comments []string
	for _, comment := range doc.List {
		comments = append(comments, strings.TrimSpace(strings.TrimPrefix(comment.Text, "//")))
	}
	return strings.Join(comments, " ")
}

func extractFuncType(f *ast.FuncType) ([]ParameterInfo, string) {
	var params []ParameterInfo
	var returnType string

	// Extract parameters
	if f.Params != nil && f.Params.List != nil {
		for _, p := range f.Params.List {
			typeName := ""
			typeName = exprToString(p.Type)

			// Handle multiple names in the same parameter declaration
			if len(p.Names) > 0 {
				for _, name := range p.Names {
					params = append(params, ParameterInfo{
						Name: name.Name,
						Type: typeName,
					})
				}
			} else {
				// For unnamed parameters (like interface methods)
				params = append(params, ParameterInfo{
					Name: "",
					Type: typeName,
				})
			}
		}
	}

	// Extract return type
	if f.Results != nil && f.Results.List != nil {
		var returns []string
		for _, r := range f.Results.List {
			typeName := exprToString(r.Type)
			returns = append(returns, typeName)
		}
		if len(returns) == 1 {
			returnType = returns[0]
		} else if len(returns) > 1 {
			returnType = "(" + strings.Join(returns, ", ") + ")"
		} else {
			returnType = "void"
		}
	} else {
		returnType = "void"
	}

	return params, returnType
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		params, returnType := extractFuncType(t)
		var paramStrings []string
		for _, p := range params {
			if p.Name != "" {
				paramStrings = append(paramStrings, p.Name+" "+p.Type)
			} else {
				paramStrings = append(paramStrings, p.Type)
			}
		}
		return "func(" + strings.Join(paramStrings, ", ") + ") " + returnType
	case *ast.BasicLit:
		return t.Value
	case *ast.StructType:
		return "struct{...}"
	case *ast.Ellipsis:
		return "..." + exprToString(t.Elt)
	case *ast.ChanType:
		return "chan " + exprToString(t.Value)
	default:
		return fmt.Sprintf("<%T>", expr)
	}
}

func extractStructMethods(pkg *ast.Package, structName string) ([]FunctionInfo, map[string]string) {
	var methods []FunctionInfo
	structMethodsMap := make(map[string]string)

	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				recvType := funcDecl.Recv.List[0].Type
				var typeName string

				// Check if it's a pointer receiver
				if starExpr, ok := recvType.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						typeName = ident.Name
					}
				} else if ident, ok := recvType.(*ast.Ident); ok {
					typeName = ident.Name
				}

				if typeName == structName {
					// Generate a unique ID for this method
					methodID := generateID("func_")
					params, returnType := extractFuncType(funcDecl.Type)
					methodInfo := FunctionInfo{
						Name:       funcDecl.Name.Name,
						Parameters: params,
						ReturnType: returnType,
						Comment:    extractComment(funcDecl.Doc),
						ID:         methodID,
					}
					methods = append(methods, methodInfo)

					// Store method ID
					fullMethodName := structName + "." + funcDecl.Name.Name
					funcMap[fullMethodName] = methodID
					structMethodsMap[funcDecl.Name.Name] = methodID
				}
			}
			return true
		})
	}

	return methods, structMethodsMap
}

// processGoFile analyzes a single Go file and extracts its structure
func processGoFile(filePath, projectName, packageName string) (ModuleInfo, error) {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return ModuleInfo{}, err
	}

	moduleInfo := ModuleInfo{
		Structs:      []StructInfo{},
		Functions:    []FunctionInfo{},
		Interfaces:   []InterfaceInfo{},
		Dependencies: []string{},
		Constants:    []ConstantInfo{},
		Variables:    []VariableInfo{},
	}

	// Extract imports
	for _, imp := range node.Imports {
		path := imp.Path.Value
		var name string
		if imp.Name != nil {
			name = imp.Name.Name + " "
		}
		moduleInfo.Dependencies = append(moduleInfo.Dependencies, "import "+name+path)
	}

	// Process declarations
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Skip methods (they'll be handled with structs)
			if d.Recv == nil {
				// Regular function, not a method
				funcID := generateID("func_")
				params, returnType := extractFuncType(d.Type)
				funcInfo := FunctionInfo{
					Name:       d.Name.Name,
					Parameters: params,
					ReturnType: returnType,
					Comment:    extractComment(d.Doc),
					ID:         funcID,
					Package:    packageName,
					FilePath:   filePath,
				}
				moduleInfo.Functions = append(moduleInfo.Functions, funcInfo)

				// Register the function ID
				fullFuncName := packageName + "." + d.Name.Name
				funcMap[fullFuncName] = funcID
				funcMap[d.Name.Name] = funcID // Also register just the name for local references

				// Add to nodes
				nodes[funcID] = Node{
					ID:      funcID,
					Type:    "function",
					Name:    d.Name.Name,
					Package: packageName,
					File:    filepath.Base(filePath),
				}

				// Analyze function body for calls to other functions
				if d.Body != nil {
					ast.Inspect(d.Body, func(n ast.Node) bool {
						if callExpr, ok := n.(*ast.CallExpr); ok {
							detectFunctionCall(callExpr, funcID, packageName)
						}
						// Look for type usage in declarations
						if declStmt, ok := n.(*ast.DeclStmt); ok {
							if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok {
								processGenDeclForTypeUsage(genDecl, funcID)
							}
						}
						// Look for type usage in assignments
						if assignStmt, ok := n.(*ast.AssignStmt); ok {
							for _, rhs := range assignStmt.Rhs {
								if compLit, ok := rhs.(*ast.CompositeLit); ok {
									if ident, ok := compLit.Type.(*ast.Ident); ok {
										if typeID, exists := typeMap[ident.Name]; exists {
											edges = append(edges, Edge{
												From:     funcID,
												To:       typeID,
												Relation: "uses",
											})
										} else if structID, exists := structMap[ident.Name]; exists {
											edges = append(edges, Edge{
												From:     funcID,
												To:       structID,
												Relation: "instantiates",
											})
										}
									}
								}
							}
						}
						return true
					})
				}
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					// Handle struct types
					if structType, ok := s.Type.(*ast.StructType); ok {
						structID := generateID("struct_")
						structInfo := StructInfo{
							Name:       s.Name.Name,
							Properties: []PropertyInfo{},
							Comment:    extractComment(d.Doc),
							ID:         structID,
						}

						// Register struct ID
						structMap[s.Name.Name] = structID
						typeMap[s.Name.Name] = structID

						// Add to nodes
						nodes[structID] = Node{
							ID:      structID,
							Type:    "struct",
							Name:    s.Name.Name,
							Package: packageName,
							File:    filepath.Base(filePath),
						}

						// Extract struct fields
						if structType.Fields != nil {
							for _, field := range structType.Fields.List {
								if len(field.Names) > 0 {
									for _, name := range field.Names {
										typeName := exprToString(field.Type)
										structInfo.Properties = append(structInfo.Properties, PropertyInfo{
											Name:    name.Name,
											Type:    typeName,
											Comment: extractComment(field.Doc),
										})

										// Check if field type references another struct/type
										if typeID, exists := typeMap[typeName]; exists {
											edges = append(edges, Edge{
												From:     structID,
												To:       typeID,
												Relation: "has_field_of_type",
											})
										}
									}
								} else {
									// Embedded field
									fieldType := exprToString(field.Type)
									structInfo.Properties = append(structInfo.Properties, PropertyInfo{
										Name:    fieldType, // The name is the type for embedded fields
										Type:    fieldType,
										Comment: extractComment(field.Doc),
									})

									// Add relationship for embedded struct
									if typeID, exists := typeMap[fieldType]; exists {
										edges = append(edges, Edge{
											From:     structID,
											To:       typeID,
											Relation: "embeds",
										})
									}
								}
							}
						}

						// Find methods for this struct (will be populated later)
						moduleInfo.Structs = append(moduleInfo.Structs, structInfo)
					}

					// Handle interfaces
					if interfaceType, ok := s.Type.(*ast.InterfaceType); ok {
						interfaceID := generateID("interface_")
						interfaceInfo := InterfaceInfo{
							Name:      s.Name.Name,
							Functions: []FunctionInfo{},
							Comment:   extractComment(d.Doc),
							ID:        interfaceID,
						}

						// Register interface ID
						typeMap[s.Name.Name] = interfaceID

						// Add to nodes
						nodes[interfaceID] = Node{
							ID:      interfaceID,
							Type:    "interface",
							Name:    s.Name.Name,
							Package: packageName,
							File:    filepath.Base(filePath),
						}

						// Extract interface methods
						if interfaceType.Methods != nil {
							for _, method := range interfaceType.Methods.List {
								if len(method.Names) > 0 {
									if methodType, ok := method.Type.(*ast.FuncType); ok {
										params, returnType := extractFuncType(methodType)
										methodID := generateID("method_")
										for _, name := range method.Names {
											methodInfo := FunctionInfo{
												Name:       name.Name,
												Parameters: params,
												ReturnType: returnType,
												Comment:    extractComment(method.Doc),
												ID:         methodID,
											}
											interfaceInfo.Functions = append(interfaceInfo.Functions, methodInfo)

											// Add method to nodes
											nodes[methodID] = Node{
												ID:      methodID,
												Type:    "interface_method",
												Name:    name.Name,
												Package: packageName,
												File:    filepath.Base(filePath),
											}

											// Add relationship between interface and method
											edges = append(edges, Edge{
												From:     interfaceID,
												To:       methodID,
												Relation: "declares",
											})
										}
									}
								}
							}
						}

						moduleInfo.Interfaces = append(moduleInfo.Interfaces, interfaceInfo)
					}

				case *ast.ValueSpec:
					// Handle constants and variables
					if d.Tok == token.CONST {
						for i, name := range s.Names {
							constID := generateID("const_")
							constInfo := ConstantInfo{
								Name: name.Name,
								Type: "",
								ID:   constID,
							}

							// Add to nodes
							nodes[constID] = Node{
								ID:      constID,
								Type:    "constant",
								Name:    name.Name,
								Package: packageName,
								File:    filepath.Base(filePath),
							}

							if s.Type != nil {
								constInfo.Type = exprToString(s.Type)

								// Check if constant type references another type
								if typeID, exists := typeMap[constInfo.Type]; exists {
									edges = append(edges, Edge{
										From:     constID,
										To:       typeID,
										Relation: "has_type",
									})
								}
							}

							if i < len(s.Values) {
								constInfo.Value = exprToString(s.Values[i])
							}

							moduleInfo.Constants = append(moduleInfo.Constants, constInfo)
						}
					} else if d.Tok == token.VAR {
						for i, name := range s.Names {
							varID := generateID("var_")
							varInfo := VariableInfo{
								Name: name.Name,
								Type: "",
								ID:   varID,
							}

							// Add to nodes
							nodes[varID] = Node{
								ID:      varID,
								Type:    "variable",
								Name:    name.Name,
								Package: packageName,
								File:    filepath.Base(filePath),
							}

							if s.Type != nil {
								varInfo.Type = exprToString(s.Type)

								// Check if variable type references another type
								if typeID, exists := typeMap[varInfo.Type]; exists {
									edges = append(edges, Edge{
										From:     varID,
										To:       typeID,
										Relation: "has_type",
									})
								}
							}

							if i < len(s.Values) {
								varInfo.Value = exprToString(s.Values[i])
							}

							moduleInfo.Variables = append(moduleInfo.Variables, varInfo)
						}
					}
				}
			}
		}
	}

	// Populate struct methods
	for i, structInfo := range moduleInfo.Structs {
		// Create a temporary package to process
		tempPkg := &ast.Package{
			Name:  "temp",
			Files: map[string]*ast.File{filePath: node},
		}
		methodsInfo, methodsMap := extractStructMethods(tempPkg, structInfo.Name)
		moduleInfo.Structs[i].Functions = methodsInfo

		// Add relationships between struct and its methods
		structID := structInfo.ID
		for methodName, methodID := range methodsMap {
			edges = append(edges, Edge{
				From:     structID,
				To:       methodID,
				Relation: "has_method",
			})

			// Also analyze method bodies for function calls
			ast.Inspect(node, func(n ast.Node) bool {
				if funcDecl, ok := n.(*ast.FuncDecl); ok &&
					funcDecl.Recv != nil &&
					len(funcDecl.Recv.List) > 0 &&
					funcDecl.Name.Name == methodName {

					if funcDecl.Body != nil {
						ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
							if callExpr, ok := n.(*ast.CallExpr); ok {
								detectFunctionCall(callExpr, methodID, packageName)
							}
							return true
						})
					}
				}
				return true
			})
		}
	}

	return moduleInfo, nil
}

// detectFunctionCall analyzes a function call expression and adds edges for function relationships
func detectFunctionCall(callExpr *ast.CallExpr, callerID string, packageName string) {
	switch fun := callExpr.Fun.(type) {
	case *ast.Ident:
		// Local function call
		if calleeID, exists := funcMap[fun.Name]; exists {
			edges = append(edges, Edge{
				From:     callerID,
				To:       calleeID,
				Relation: "calls",
			})
		}
	case *ast.SelectorExpr:
		// Could be a package.Function call or object.Method call
		if x, ok := fun.X.(*ast.Ident); ok {
			// Try as package.Function
			fullName := x.Name + "." + fun.Sel.Name
			if calleeID, exists := funcMap[fullName]; exists {
				edges = append(edges, Edge{
					From:     callerID,
					To:       calleeID,
					Relation: "calls",
				})
			}

			// Or it could be a method call on a struct instance
			// This is more complex and would require type checking
		}
	}
}

// processGenDeclForTypeUsage checks for type usage in declarations
func processGenDeclForTypeUsage(genDecl *ast.GenDecl, funcID string) {
	for _, spec := range genDecl.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			if valueSpec.Type != nil {
				if ident, ok := valueSpec.Type.(*ast.Ident); ok {
					if typeID, exists := typeMap[ident.Name]; exists {
						edges = append(edges, Edge{
							From:     funcID,
							To:       typeID,
							Relation: "uses",
						})
					}
				}
			}
		}
	}
}

func processGoProject(projectPath string, projectName string) (ProjectStructure, error) {
	result := ProjectStructure{
		Project: map[string]PackageInfo{
			projectName: {
				Modules: make(map[string]ModuleInfo),
			},
		},
		CodeGraph: CodeGraph{
			Nodes: []Node{},
			Edges: []Edge{},
		},
	}

	// First pass: determine package structure and collect package info
	packagePaths := make(map[string]string) // Maps package path to package name

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") &&
			!strings.Contains(path, "/vendor/") &&
			!strings.HasSuffix(path, "_test.go") {

			// Parse file to get package name
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
			if err != nil {
				fmt.Printf("Warning: Error parsing %s: %v\n", path, err)
				return nil
			}

			dir := filepath.Dir(path)
			packagePaths[dir] = f.Name.Name
		}
		return nil
	})

	if err != nil {
		return result, err
	}

	// Second pass: process each file with knowledge of its package
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") &&
			!strings.Contains(path, "/vendor/") &&
			!strings.HasSuffix(path, "_test.go") {

			relPath, err := filepath.Rel(projectPath, path)
			if err != nil {
				return err
			}

			dir := filepath.Dir(path)
			packageName := packagePaths[dir]

			moduleInfo, err := processGoFile(path, projectName, packageName)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				return nil // Continue with other files
			}

			result.Project[projectName].Modules[relPath] = moduleInfo
		}

		return nil
	})

	// Now convert our map of nodes to a slice for JSON output
	for _, node := range nodes {
		result.CodeGraph.Nodes = append(result.CodeGraph.Nodes, node)
	}
	result.CodeGraph.Edges = edges

	return result, err
}

func main() {
	// Default values
	projectPath := "."
	projectName := "MyProject"
	outputFile := "output.json"

	// Parse command-line arguments
	switch len(os.Args) {
	case 4:
		outputFile = os.Args[3]
		fallthrough
	case 3:
		projectName = os.Args[2]
		fallthrough
	case 2:
		projectPath = os.Args[1]
	case 1:
		// Use defaults
	default:
		fmt.Println("Usage: go run go-codegraph.go [project-path] [project-name] [output-file]")
		os.Exit(1)
	}

	// Convert to absolute path
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error converting to absolute path: %v\n", err)
		os.Exit(1)
	}

	result, err := processGoProject(absProjectPath, projectName)
	if err != nil {
		fmt.Printf("Error processing project: %v\n", err)
		os.Exit(1)
	}

	// Generate JSON
	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	err = os.WriteFile(outputFile, jsonOutput, 0644)
	if err != nil {
		fmt.Printf("Error writing to output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analysis complete. Results written to %s\n", outputFile)
}
