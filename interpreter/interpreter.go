package interpreter

import (
	"fmt"
	"github.com/fadion/aria/ast"
	"github.com/fadion/aria/reporter"
	"math"
	"strings"
	"io/ioutil"
	"github.com/fadion/aria/lexer"
	"github.com/fadion/aria/reader"
	"github.com/fadion/aria/parser"
	"path/filepath"
)

// Interpreter represents the interpreter.
type Interpreter struct {
	modules     map[string]*ModuleType
	library     *Library
	functions   map[string]string
	moduleCache map[string]*Scope
	importCache map[string]*ast.Program
}

// New initializes an Interpreter.
func New() *Interpreter {
	lib := NewLibrary()
	lib.Register()

	return &Interpreter{
		modules:     map[string]*ModuleType{},
		library:     lib,
		functions:   map[string]string{},
		moduleCache: map[string]*Scope{},
		importCache: map[string]*ast.Program{},
	}
}

// Interpret runs the interpreter.
func (i *Interpreter) Interpret(node ast.Node, scope *Scope) DataType {
	switch node := node.(type) {
	case *ast.Program:
		return i.runProgram(node, scope)
	case *ast.Module:
		return i.runModule(node, scope)
	case *ast.ModuleAccess:
		return i.runModuleAccess(node, scope)
	case *ast.Identifier:
		return i.runIdentifier(node, scope)
	case *ast.Let:
		return i.runLet(node, scope)
	case *ast.String:
		return &StringType{Value: node.Value}
	case *ast.Atom:
		return &AtomType{Value: node.Value}
	case *ast.Integer:
		return &IntegerType{Value: node.Value}
	case *ast.Float:
		return &FloatType{Value: node.Value}
	case *ast.Boolean:
		return i.nativeToBoolean(node.Value)
	case *ast.Array:
		return i.runArray(node, scope)
	case *ast.Dictionary:
		return i.runDictionary(node, scope)
	case *ast.ExpressionStatement:
		return i.Interpret(node.Expression, scope)
	case *ast.BlockStatement:
		return i.runBlockStatement(node, scope)
	case *ast.PrefixExpression:
		return i.runPrefix(node, scope)
	case *ast.InfixExpression:
		return i.runInfix(node, scope)
	case *ast.Pipe:
		return i.runPipe(node, scope)
	case *ast.If:
		return i.runIf(node, scope)
	case *ast.Switch:
		return i.runSwitch(node, scope)
	case *ast.For:
		return i.runFor(node, scope)
	case *ast.Function:
		return &FunctionType{
			Parameters: node.Parameters.Elements,
			Body:       node.Body,
			Scope:      NewScope(),
		}
	case *ast.FunctionCall:
		return i.runFunction(node, scope)
	case *ast.Import:
		return i.runImport(node, scope)
	case *ast.Subscript:
		return i.runSubscript(node, scope)
	case *ast.Return:
		return &ReturnType{Value: i.Interpret(node.Value, scope)}
	case *ast.Break:
		return &BreakType{}
	case *ast.Continue:
		return &ContinueType{}
	case *ast.Placeholder:
		return &PlaceholderType{}
	}

	return nil
}

// Interpret the program statement by statement.
func (i *Interpreter) runProgram(node *ast.Program, scope *Scope) DataType {
	var result DataType

	for _, statement := range node.Statements {
		result = i.Interpret(statement, scope)
	}

	return result
}

// Interpret a Module.
func (i *Interpreter) runModule(node *ast.Module, scope *Scope) DataType {
	if _, ok := i.modules[node.Name.Value]; ok {
		i.reportError(node, fmt.Sprintf("Module '%s' redeclared", node.Name.Value))
	} else {
		// Store the module name and DataType for easier
		// reference later.
		i.modules[node.Name.Value] = &ModuleType{Name: node.Name, Body: node.Body}
	}

	return nil
}

// Interpret Module access.
func (i *Interpreter) runModuleAccess(node *ast.ModuleAccess, scope *Scope) DataType {
	scope = NewScope()

	// Check if the module exists.
	if module, ok := i.modules[node.Object.Value]; ok {
		// Check the cache for already interpreted properties.
		// Otherwise run them and pass their values to the scope.
		if sc, ok := i.moduleCache[module.Name.Value]; ok {
			scope = sc
		} else {
			i.runModuleProperties(module.Body, scope)
			i.moduleCache[module.Name.Value] = scope
		}

		for _, statement := range module.Body.Statements {
			switch sType := statement.(type) {
			case *ast.Let: // All module statements should be LET.
				if sType.Name.Value == node.Parameter.Value {
					switch value := sType.Value.(type) {
					case *ast.Function:
						// In case of a function, return the FunctionType
						// with the current scope set.
						return &FunctionType{
							Parameters: value.Parameters.Elements,
							Body:       value.Body,
							Scope:      scope,
						}
					default:
						// Any other value is interpretet and returned.
						// These are like constants.
						return i.Interpret(value, scope)
					}
				}
			default:
				i.reportError(node, "Only LET statements are accepted as Module members")
				return nil
			}
		}
	}

	i.reportError(node, fmt.Sprintf("Member '%s' in Module '%s' not found", node.Parameter.Value, node.Object.Value))
	return nil
}

// Interpret module properties.
func (i *Interpreter) runModuleProperties(node *ast.BlockStatement, scope *Scope) {
	for _, statement := range node.Statements {
		i.Interpret(statement, scope)
	}
}

// Interpret a let statement.
func (i *Interpreter) runLet(node *ast.Let, scope *Scope) DataType {
	object := i.Interpret(node.Value, scope)

	// On empty value, return before saving
	// the variable into the scope.
	if object == nil {
		return nil
	}

	// Save the function name and result for
	// later reference.
	switch object.(type) {
	case *FunctionType:
		i.functions[object.Inspect()] = node.Name.Value
	}

	// Check if the variable has been already
	// declared.
	if _, ok := scope.Read(node.Name.Value); ok {
		i.reportError(node, fmt.Sprintf("Identifier '%s' already declared", node.Name.Value))
		return nil
	}
	scope.Write(node.Name.Value, object)

	return object
}

// Interpret an identifier.
func (i *Interpreter) runIdentifier(node *ast.Identifier, scope *Scope) DataType {
	// Check the scope if the identifier exists.
	if object, ok := scope.Read(node.Value); ok {
		return object
	}

	i.reportError(node, fmt.Sprintf("Identifier '%s' not found in current scope", node.Value))

	return nil
}

// Interpret a block of statements.
func (i *Interpreter) runBlockStatement(node *ast.BlockStatement, scope *Scope) DataType {
	var result DataType

	// Interpret every statement of the block.
	for _, statement := range node.Statements {
		result = i.Interpret(statement, scope)
		if result == nil {
			return nil
		}

		// Check if it's one of the statements,
		// like RETURN, that should break and return
		// immediately.
		if i.shouldBreakImmediately(result) {
			return result
		}
	}

	return result
}

// Interpret an array.
func (i *Interpreter) runArray(node *ast.Array, scope *Scope) DataType {
	var result []DataType

	for _, element := range node.List.Elements {
		value := i.Interpret(element, scope)
		result = append(result, value)
	}

	return &ArrayType{Elements: result}
}

// Interpret a dictionary.
func (i *Interpreter) runDictionary(node *ast.Dictionary, scope *Scope) DataType {
	result := map[*StringType]DataType{}

	for k, v := range node.Pairs {
		key := i.Interpret(k, scope)
		switch keyObj := key.(type) {
		case *StringType: // Keys should be String only.
			value := i.Interpret(v, scope)
			result[keyObj] = value
		default:
			i.reportError(node, "Dictionaries support String keys only")
			return nil
		}
	}

	return &DictionaryType{Pairs: result}
}

// Interpret an if/then/else expression.
func (i *Interpreter) runIf(node *ast.If, scope *Scope) DataType {
	condition := i.Interpret(node.Condition, scope)

	if i.isTruthy(condition) {
		return i.Interpret(node.Then, NewScopeFrom(scope))
	} else if node.Else != nil {
		return i.Interpret(node.Else, NewScopeFrom(scope))
	} else {
		return NIL
	}
}

// Interpret a Switch expression.
func (i *Interpreter) runSwitch(node *ast.Switch, scope *Scope) DataType {
	var control DataType
	// When the control expression is missing, the Switch
	// acts as a structured if/else with a TRUE as control.
	if node.Control == nil {
		control = TRUE
	} else {
		control = i.Interpret(node.Control, scope)
		// Control expression failed.
		if control == nil {
			i.reportError(node, "Switch control expression couldn't be interpreted")
			return nil
		}
	}

	// Find the winning switch case.
	thecase, err := i.runSwitchCase(node.Cases, control, scope)
	if err != nil {
		i.reportError(node, err.Error())
		return nil
	}

	if thecase != nil {
		return i.Interpret(thecase.Body, NewScopeFrom(scope))
	}

	// Run the default case only if no winning
	// case was found.
	if node.Default != nil {
		return i.Interpret(node.Default, NewScopeFrom(scope))
	}

	return nil
}

// Interpret Switch cases by finding the winning case.
func (i *Interpreter) runSwitchCase(cases []*ast.SwitchCase, control DataType, scope *Scope) (*ast.SwitchCase, error) {
	// Iterate the switch cases.
	for _, sc := range cases {
		matches := 0
		// Iterate every parameter of the case.
		for idx, p := range sc.Values.Elements {
			parameter := i.Interpret(p, scope)

			switch {
			case parameter.Type() == control.Type():
				// Same type and same exact value.
				if parameter.Inspect() == control.Inspect() {
					return sc, nil
				}
			case parameter.Type() == ATOM_TYPE && control.Type() == STRING_TYPE:
				// A string switch can have atom cases.
				if parameter.(*AtomType).Value == control.(*StringType).Value {
					return sc, nil
				}
			case control.Type() == ARRAY_TYPE:
				arrayObj := control.(*ArrayType).Elements
				// The number of matching elements should be
				// the same as the number of array elements.
				if len(sc.Values.Elements) != len(arrayObj) {
					break
				}

				// Match found only if of the same type, same value
				// or it's a placeholder.
				if parameter.Type() == arrayObj[idx].Type() && parameter.Inspect() == arrayObj[idx].Inspect() ||
					parameter.Type() == PLACEHOLDER_TYPE {
					matches++
					// Case wins only if all the parameters match
					// all the elements of the array.
					if matches == len(arrayObj) {
						return sc, nil
					}
				}
			default:
				return nil, fmt.Errorf("Type '%s' can't be used in a Switch case with control type '%s'", parameter.Type(), control.Type())
			}
		}
	}

	return nil, nil
}

// Interpret a For expression.
func (i *Interpreter) runFor(node *ast.For, scope *Scope) DataType {
	enumObj := i.Interpret(node.Enumerable, scope)
	if enumObj == nil {
		return nil
	}

	// For in loops are valid only for iteratables:
	// Arrays, Dictionaries and Strings.
	switch enum := enumObj.(type) {
	case *ArrayType:
		return i.runForArray(node, enum, NewScopeFrom(scope))
	case *DictionaryType:
		return i.runForDictionary(node, enum, NewScopeFrom(scope))
	case *StringType:
		// Convert the string to an array so it can
		// be interpreted with the same function.
		return i.runForArray(node, i.stringToArray(enum), NewScopeFrom(scope))
	case *AtomType:
		// Treat the atom as a string.
		str := &StringType{Value: enum.Value}
		return i.runForArray(node, i.stringToArray(str), NewScopeFrom(scope))
	default:
		i.reportError(node, fmt.Sprintf("Type %s is not an enumerable", enumObj.Type()))
		return nil
	}
}

// Interpret a FOR IN Array expression.
func (i *Interpreter) runForArray(node *ast.For, array *ArrayType, scope *Scope) DataType {
	out := []DataType{}

	for idx, v := range array.Elements {
		// A single arguments gets only the current loop value.
		// Two arguments get both the key and value.
		switch len(node.Arguments.Elements) {
		case 1:
			scope.Write(node.Arguments.Elements[0].Value, v)
		case 2:
			scope.Write(node.Arguments.Elements[0].Value, &IntegerType{Value: int64(idx)})
			scope.Write(node.Arguments.Elements[1].Value, v)
		default:
			i.reportError(node, "A FOR loop with an Array expects at most 2 arguments")
			return nil
		}

		result := i.Interpret(node.Body, scope)
		// Close the loop immediately, so it doesn't report
		// multiple of the same possible error.
		if result == nil {
			return nil
		}

		// Handle break and continue keywords.
		if result.Type() == BREAK_TYPE {
			break
		} else if result.Type() == CONTINUE_TYPE {
			continue
		}

		out = append(out, result)
	}

	return &ArrayType{Elements: out}
}

// Interpret a FOR IN Dictionary expression.
func (i *Interpreter) runForDictionary(node *ast.For, dictionary *DictionaryType, scope *Scope) DataType {
	out := []DataType{}

	for k, v := range dictionary.Pairs {
		// A single arguments get the current loop value.
		// Two arguments get the key and value.
		switch len(node.Arguments.Elements) {
		case 1:
			scope.Write(node.Arguments.Elements[0].Value, v)
		case 2:
			scope.Write(node.Arguments.Elements[0].Value, &StringType{Value: k.Value})
			scope.Write(node.Arguments.Elements[1].Value, v)
		default:
			i.reportError(node, "A FOR loop with a Dictionary expects at most 2 arguments")
			return nil
		}

		result := i.Interpret(node.Body, scope)
		if result == nil {
			return nil
		}

		// Handle break and continue keywords.
		if result.Type() == BREAK_TYPE {
			break
		} else if result.Type() == CONTINUE_TYPE {
			continue
		}

		out = append(out, result)
	}

	return &ArrayType{Elements: out}
}

// Interpret a function call.
func (i *Interpreter) runFunction(node *ast.FunctionCall, scope *Scope) DataType {
	var fn DataType

	// ModuleAccess is handled differently from
	// regular functions calls.
	switch nodeType := node.Function.(type) {
	case *ast.ModuleAccess:
		// Standard library functions use the same dot
		// notation as module access. Check if the caller
		// is a standard library function first.
		if libFunc, ok := i.library.Get(nodeType.Object.Value + "." + nodeType.Parameter.Value); ok {
			// Return immediately with a value. No need for
			// further calculation.
			return i.runLibraryFunction(node, libFunc, scope)
		}

		fn = i.Interpret(nodeType, scope)
	default:
		fn = i.Interpret(nodeType, scope)
	}

	// An error, most probably on ModuleAccess, so return
	// early to stop any runtime panic.
	if fn == nil {
		return nil
	}

	// Make sure it's a function we're calling.
	if fn.Type() != FUNCTION_TYPE {
		i.reportError(node, "Trying to call a non-function")
		return nil
	}

	function := fn.(*FunctionType)
	arguments := []DataType{}

	// Check for arguments/parameters missmatch.
	if len(node.Arguments.Elements) > len(function.Parameters) {
		i.reportError(node, "Too many arguments in function call")
		return nil
	} else if len(node.Arguments.Elements) < len(function.Parameters) {
		i.reportError(node, "Too few arguments in function call")
		return nil
	}

	// Write the function's name to its scope so
	// it can reference itself. If it's not found
	// in the "functions" map, it means it's a function
	// not declared with a "let" statement.
	if v, ok := i.functions[fn.Inspect()]; ok {
		function.Scope.Write(v, function)
	}

	// Interpret every single argument and pass it
	// to the function's scope.
	for index, element := range node.Arguments.Elements {
		value := i.Interpret(element, scope)
		if value != nil {
			arguments = append(arguments, value)
			function.Scope.Write(function.Parameters[index].Value, value)
		}
	}

	var result DataType

	// This could by run by "runBlockStatement", but it
	// needs to check for closures.
	for _, statement := range function.Body.Statements {
		result = i.Interpret(statement, function.Scope)
		if result == nil {
			return nil
		}

		// A function is found inside the current function.
		// Closure or not, pass the scope so it can access
		// the parent's arguments and variables.
		if result.Type() == FUNCTION_TYPE {
			result.(*FunctionType).Scope.Merge(function.Scope)
		}

		if i.shouldBreakImmediately(result) {
			break
		}
	}

	return i.unwrapReturnValue(result)
}

// Run a function from the Standard Library.
func (i *Interpreter) runLibraryFunction(node *ast.FunctionCall, libFunc libraryFunc, scope *Scope) DataType {
	args := []DataType{}
	// Interpret all the arguments and pass them
	// as objects to the array.
	for _, element := range node.Arguments.Elements {
		value := i.Interpret(element, scope)
		if value != nil {
			args = append(args, value)
		}
	}

	// Execute the library function.
	// libFunc is a function received from
	// Library.get().
	libObject, err := libFunc(args...)
	if err != nil {
		i.reportError(node, err.Error())
		return nil
	}

	return libObject
}

// Interpret an Array or Dictionary index call.
func (i *Interpreter) runSubscript(node *ast.Subscript, scope *Scope) DataType {
	left := i.Interpret(node.Left, scope)
	index := i.Interpret(node.Index, scope)

	// No point in continuing if any of the values
	// is nil.
	if left == nil || index == nil {
		return nil
	}

	switch {
	case left.Type() == ARRAY_TYPE && index.Type() == INTEGER_TYPE:
		result, err := i.runArraySubscript(left, index)
		if err != nil {
			i.reportError(node, err.Error())
		}
		return result
	case left.Type() == DICTIONARY_TYPE && index.Type() == STRING_TYPE:
		result, err := i.runDictionarySubscript(left, index)
		if err != nil {
			i.reportError(node, err.Error())
		}
		return result
	case left.Type() == STRING_TYPE && index.Type() == INTEGER_TYPE:
		result, err := i.runStringSubscript(left, index)
		if err != nil {
			i.reportError(node, err.Error())
		}
		return result
	case left.Type() == ATOM_TYPE && index.Type() == INTEGER_TYPE:
		str := &StringType{Value: left.(*AtomType).Value}
		result, err := i.runStringSubscript(str, index)
		if err != nil {
			i.reportError(node, err.Error())
		}
		return result
	default:
		i.reportError(node, fmt.Sprintf("Subscript on '%s' not supported with literal '%s'", left.Type(), index.Type()))
		return nil
	}
}

// Interpret an Array subscript.
func (i *Interpreter) runArraySubscript(array, index DataType) (DataType, error) {
	arrayObj := array.(*ArrayType).Elements
	idx := index.(*IntegerType).Value
	originalIdx := idx

	// Negative index accesses elements starting
	// from the right side of the array.
	if idx < 0 {
		idx = int64(len(arrayObj)) + idx
	}

	// Check bounds.
	if idx < 0 || idx > int64(len(arrayObj)-1) {
		return nil, fmt.Errorf("Array index '%d' out of bounds", originalIdx)
	}

	return arrayObj[idx], nil
}

// Interpret a Dictionary subscript.
func (i *Interpreter) runDictionarySubscript(dictionary, index DataType) (DataType, error) {
	dictObj := dictionary.(*DictionaryType).Pairs
	key := index.(*StringType).Value

	for k, v := range dictObj {
		if k.Value == key {
			return v, nil
		}
	}

	return nil, fmt.Errorf("Key '%s' doesn't exist in Dictionary", key)
}

// Interpret a String subscript.
func (i *Interpreter) runStringSubscript(str, index DataType) (DataType, error) {
	arrayObj := str.(*StringType).Value
	idx := index.(*IntegerType).Value

	// Check bounds.
	if idx < 0 || idx > int64(len(arrayObj)-1) {
		return nil, fmt.Errorf("String index '%d' out of bounds", idx)
	}

	return &StringType{Value: string(arrayObj[idx])}, nil
}

// Interpret Pipe operator: FUNCTION_CALL() |> FUNCTION_CALL()
func (i *Interpreter) runPipe(node *ast.Pipe, scope *Scope) DataType {
	left := i.Interpret(node.Left, scope)
	// Convert the type object back to an expression
	// so it can be passed to the FunctionCall arguments.
	argument := i.typeToExpression(left)
	if argument == nil {
		return nil
	}

	// The right side operator should be a function.
	switch rightFunc := node.Right.(type) {
	case *ast.FunctionCall:
		// Prepend the left-hand interpreted value
		// to the function arguments.
		rightFunc.Arguments.Elements = append([]ast.Expression{argument}, rightFunc.Arguments.Elements...)
		return i.Interpret(rightFunc, scope)
	}

	return nil
}

// Import "filename" by reading, lexing and
// parsing it all over.
func (i *Interpreter) runImport(node *ast.Import, scope *Scope) DataType {
	filename := i.prepareImportFilename(node.File.Value)

	// Check the cache fist.
	if cache, ok := i.importCache[filename]; ok {
		return i.Interpret(cache, scope)
	}

	source, err := ioutil.ReadFile(i.prepareImportFilename(filename))
	if err != nil {
		i.reportError(node, fmt.Sprintf("Couldn't read imported file '%s'", node.File.Value))
		return nil
	}

	lex := lexer.New(reader.New(source))
	if reporter.HasErrors() {
		return nil
	}

	parse := parser.New(lex)
	program := parse.Parse()
	if reporter.HasErrors() {
		return nil
	}

	// Cache the parsed program.
	i.importCache[filename] = program

	return i.Interpret(program, scope)
}

// Interpret prefix operators: (OP)OBJ
func (i *Interpreter) runPrefix(node *ast.PrefixExpression, scope *Scope) DataType {
	object := i.Interpret(node.Right, scope)

	if object == nil {
		i.reportError(node, fmt.Sprintf("Trying to run operator '%s' with an unknown value", node.Operator))
		return nil
	}

	var out DataType
	var err error

	switch node.Operator {
	case "!": // !true or !0
		out = i.nativeToBoolean(!i.isTruthy(object))
	case "-": // -5
		out, err = i.runMinusPrefix(object)
	case "~": // ~9
		out, err = i.runBitwiseNotPrefix(object)
	default:
		err = fmt.Errorf("Unsupported prefix operator")
	}

	if err != nil {
		i.reportError(node, err.Error())
	}

	return out
}

// - prefix operator.
func (i *Interpreter) runMinusPrefix(object DataType) (DataType, error) {
	switch object.Type() {
	case INTEGER_TYPE:
		return &IntegerType{Value: -object.(*IntegerType).Value}, nil
	case FLOAT_TYPE:
		return &FloatType{Value: -object.(*FloatType).Value}, nil
	default:
		return nil, fmt.Errorf("Minus prefix can be applied to Integers and Floats only")
	}
}

// ~ prefix operator.
func (i *Interpreter) runBitwiseNotPrefix(object DataType) (DataType, error) {
	switch object.Type() {
	case INTEGER_TYPE:
		return &IntegerType{Value: ^object.(*IntegerType).Value}, nil
	default:
		return nil, fmt.Errorf("Bitwise NOT prefix can be applied to Integers only")
	}
}

// Interpret infix operators: LEFT (OP) RIGHT
func (i *Interpreter) runInfix(node *ast.InfixExpression, scope *Scope) DataType {
	left := i.Interpret(node.Left, scope)
	right := i.Interpret(node.Right, scope)

	if left == nil || right == nil {
		i.reportError(node, fmt.Sprintf("Trying to run operator '%s' with unknown value", node.Operator))
		return nil
	}

	var out DataType
	var err error

	// Infix operators have different meaning for different
	// data types. Every possible combination of data type
	// is checked and run in its own function.
	switch {
	case left.Type() == INTEGER_TYPE && right.Type() == INTEGER_TYPE:
		out, err = i.runIntegerInfix(node.Operator, left, right)
	case left.Type() == FLOAT_TYPE && right.Type() == FLOAT_TYPE:
		out, err = i.runFloatInfix(node.Operator, left.(*FloatType).Value, right.(*FloatType).Value)
	case left.Type() == FLOAT_TYPE && right.Type() == INTEGER_TYPE:
		// Treat the integer as a float to allow
		// operations between the two.
		out, err = i.runFloatInfix(node.Operator, left.(*FloatType).Value, float64(right.(*IntegerType).Value))
	case left.Type() == INTEGER_TYPE && right.Type() == FLOAT_TYPE:
		// Same as above: treat the integer as a float.
		out, err = i.runFloatInfix(node.Operator, float64(left.(*IntegerType).Value), right.(*FloatType).Value)
	case left.Type() == STRING_TYPE && right.Type() == STRING_TYPE:
		out, err = i.runStringInfix(node.Operator, left.(*StringType).Value, right.(*StringType).Value)
	case left.Type() == ATOM_TYPE && right.Type() == ATOM_TYPE:
		// Treat atoms as string.
		out, err = i.runStringInfix(node.Operator, left.(*AtomType).Value, right.(*AtomType).Value)
	case left.Type() == ATOM_TYPE && right.Type() == STRING_TYPE:
		out, err = i.runStringInfix(node.Operator, left.(*AtomType).Value, right.(*StringType).Value)
	case left.Type() == STRING_TYPE && right.Type() == ATOM_TYPE:
		out, err = i.runStringInfix(node.Operator, left.(*StringType).Value, right.(*AtomType).Value)
	case left.Type() == BOOLEAN_TYPE && right.Type() == BOOLEAN_TYPE:
		out, err = i.runBooleanInfix(node.Operator, left, right)
	case left.Type() == ARRAY_TYPE && right.Type() == ARRAY_TYPE:
		out, err = i.runArrayInfix(node.Operator, left, right)
	case left.Type() == DICTIONARY_TYPE && right.Type() == DICTIONARY_TYPE:
		out, err = i.runDictionaryInfix(node.Operator, left, right)
	case left.Type() != right.Type():
		err = fmt.Errorf("Cannot run expression with types '%s' and '%s'", left.Type(), right.Type())
	default:
		err = fmt.Errorf("Uknown operator %s for types '%s' and '%s'", node.Operator, left.Type(), right.Type())
	}

	if err != nil {
		i.reportError(node, err.Error())
	}

	return out
}

// Interpret infix operation for Integers.
func (i *Interpreter) runIntegerInfix(operator string, left, right DataType) (DataType, error) {
	leftVal := left.(*IntegerType).Value
	rightVal := right.(*IntegerType).Value

	switch operator {
	case "+":
		return &IntegerType{Value: leftVal + rightVal}, nil
	case "-":
		return &IntegerType{Value: leftVal - rightVal}, nil
	case "*":
		return &IntegerType{Value: leftVal * rightVal}, nil
	case "/":
		value := float64(leftVal) / float64(rightVal)
		// Check if it's a full number, so it can be returned
		// as an Integer object. Otherwise it will be a Float object.
		if math.Trunc(value) == value {
			return &IntegerType{Value: int64(value)}, nil
		}

		return &FloatType{Value: value}, nil
	case "%":
		return &IntegerType{Value: leftVal % rightVal}, nil
	case "**": // Exponentiation
		return &IntegerType{Value: int64(math.Pow(float64(leftVal), float64(rightVal)))}, nil
	case "<":
		return i.nativeToBoolean(leftVal < rightVal), nil
	case "<=":
		return i.nativeToBoolean(leftVal <= rightVal), nil
	case ">":
		return i.nativeToBoolean(leftVal > rightVal), nil
	case ">=":
		return i.nativeToBoolean(leftVal >= rightVal), nil
	case "<<":
		// Shift needs two positive integers.
		if leftVal < 0 || rightVal < 0 {
			return nil, fmt.Errorf("Bitwise shift requires two unsigned Integers")
		}
		return &IntegerType{Value: int64(uint64(leftVal) << uint64(rightVal))}, nil
	case ">>":
		if leftVal < 0 || rightVal < 0 {
			return nil, fmt.Errorf("Bitwsise shift requires two unsigned Integers")
		}
		return &IntegerType{Value: int64(uint64(leftVal) >> uint64(rightVal))}, nil
	case "&":
		return &IntegerType{Value: leftVal & rightVal}, nil
	case "|":
		return &IntegerType{Value: leftVal | rightVal}, nil
	case "==":
		return i.nativeToBoolean(leftVal == rightVal), nil
	case "!=":
		return i.nativeToBoolean(leftVal != rightVal), nil
	case "..":
		return i.runRangeIntegerInfix(leftVal, rightVal), nil
	default:
		return nil, fmt.Errorf("Unsupported Integer operator '%s'", operator)
	}
}

// Interpret infix operation for Floats.
func (i *Interpreter) runFloatInfix(operator string, left, right float64) (DataType, error) {
	switch operator {
	case "+":
		return &FloatType{Value: left + right}, nil
	case "-":
		return &FloatType{Value: left - right}, nil
	case "*":
		return &FloatType{Value: left * right}, nil
	case "/":
		return &FloatType{Value: left / right}, nil
	case "%":
		return &FloatType{Value: math.Mod(left, right)}, nil
	case "**":
		return &FloatType{Value: math.Pow(left, right)}, nil
	case "<":
		return i.nativeToBoolean(left < right), nil
	case "<=":
		return i.nativeToBoolean(left <= right), nil
	case ">":
		return i.nativeToBoolean(left > right), nil
	case ">=":
		return i.nativeToBoolean(left >= right), nil
	case "==":
		return i.nativeToBoolean(left == right), nil
	case "!=":
		return i.nativeToBoolean(left != right), nil
	default:
		return nil, fmt.Errorf("Unsupported Float operator '%s'", operator)
	}
}

// Interpret infix operation for Strings.
func (i *Interpreter) runStringInfix(operator string, left, right string) (DataType, error) {
	switch operator {
	case "+": // Concat two strings.
		return &StringType{Value: left + right}, nil
	case "<":
		return i.nativeToBoolean(len(left) < len(right)), nil
	case "<=":
		return i.nativeToBoolean(len(left) <= len(right)), nil
	case ">":
		return i.nativeToBoolean(len(left) > len(right)), nil
	case ">=":
		return i.nativeToBoolean(len(left) >= len(right)), nil
	case "==":
		return i.nativeToBoolean(left == right), nil
	case "!=":
		return i.nativeToBoolean(left != right), nil
	case "..": // Range between two characters.
		return i.runRangeStringInfix(left, right)
	default:
		return nil, fmt.Errorf("Unsupported String operator '%s'", operator)
	}
}

// Interpret infix operation for Booleans.
func (i *Interpreter) runBooleanInfix(operator string, left, right DataType) (DataType, error) {
	leftVal := left.(*BooleanType).Value
	rightVal := right.(*BooleanType).Value

	switch operator {
	case "&&":
		return i.nativeToBoolean(leftVal && rightVal), nil
	case "||":
		return i.nativeToBoolean(leftVal || rightVal), nil
	case "==":
		return i.nativeToBoolean(leftVal == rightVal), nil
	case "!=":
		return i.nativeToBoolean(leftVal != rightVal), nil
	default:
		return nil, fmt.Errorf("Unsupported Boolean operator '%s'", operator)
	}
}

// Interpret infix operation for Arrays.
func (i *Interpreter) runArrayInfix(operator string, left, right DataType) (DataType, error) {
	leftVal := left.(*ArrayType).Elements
	rightVal := right.(*ArrayType).Elements

	switch operator {
	case "+": // Combine two arrays.
		return &ArrayType{Elements: append(leftVal, rightVal...)}, nil
	case "==":
		return i.nativeToBoolean(i.compareArrays(leftVal, rightVal)), nil
	case "!=":
		return i.nativeToBoolean(!i.compareArrays(leftVal, rightVal)), nil
	default:
		return nil, fmt.Errorf("Unsupported Array operator '%s'", operator)
	}
}

// Interpret infix operation for Dictionaries.
func (i *Interpreter) runDictionaryInfix(operator string, left, right DataType) (DataType, error) {
	leftVal := left.(*DictionaryType).Pairs
	rightVal := right.(*DictionaryType).Pairs

	switch operator {
	case "+": // Combine two dictionaries.
		// Add left keys to the right.
		for k, v := range leftVal {
			rightVal[k] = v
		}
		return &DictionaryType{Pairs: rightVal}, nil
	case "==":
		return i.nativeToBoolean(i.compareDictionaries(leftVal, rightVal)), nil
	case "!=":
		return i.nativeToBoolean(!i.compareDictionaries(leftVal, rightVal)), nil
	default:
		return nil, fmt.Errorf("Unsupported Dictionary operator '%s'", operator)
	}
}

// Generate an array from two integers.
func (i *Interpreter) runRangeIntegerInfix(left, right int64) DataType {
	result := []DataType{}

	if left < right {
		// 0 -> 9
		for idx := left; idx <= right; idx++ {
			result = append(result, &IntegerType{Value: idx})
		}
	} else {
		// 9 -> 0
		for idx := left; idx >= right; idx-- {
			result = append(result, &IntegerType{Value: idx})
		}
	}

	return &ArrayType{Elements: result}
}

// Generate an array from two strings.
func (i *Interpreter) runRangeStringInfix(left, right string) (DataType, error) {
	if len(left) > 1 || len(right) > 1 {
		return nil, fmt.Errorf("Range operator expects 2 single character strings")
	}

	result := []DataType{}
	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
	// Only lowercase, single character strings are supported.
	// Convert it to int32 for easy comparison in the loop.
	leftByte := []int32(strings.ToLower(left))[0]
	rightByte := []int32(strings.ToLower(right))[0]

	if leftByte < rightByte {
		// a -> z
		for _, v := range alphabet {
			if v >= leftByte && v <= rightByte {
				result = append(result, &StringType{Value: string(v)})
			}
		}
	} else {
		// z -> a
		for i := len(alphabet) - 1; i >= 0; i-- {
			v := int32(alphabet[i])
			if v <= leftByte && v >= rightByte {
				result = append(result, &StringType{Value: string(v)})
			}
		}
	}

	return &ArrayType{Elements: result}, nil
}

// Check if it's an object that triggers an immediate
// break of the block.
func (i *Interpreter) shouldBreakImmediately(object DataType) bool {
	switch object.Type() {
	case RETURN_TYPE, BREAK_TYPE, CONTINUE_TYPE:
		return true
	default:
		return false
	}
}

// Get the value from a return statement.
func (i *Interpreter) unwrapReturnValue(object DataType) DataType {
	// If it is a Return value, unwrap it. Otherwise
	// just return the original object.
	if returnVal, ok := object.(*ReturnType); ok {
		return returnVal.Value
	}

	return object
}

// Check if two arrays are identical if all of their
// elements are the same.
func (i *Interpreter) compareArrays(left, right []DataType) bool {
	if len(left) != len(right) {
		return false
	}

	for i, v := range left {
		// Same type and same string representation.
		if v.Type() != right[i].Type() || v.Inspect() != right[i].Inspect() {
			return false
		}
	}

	return true
}

// Check if two dictionaries are identical if all of their keys
// are the same.
func (i *Interpreter) compareDictionaries(left, right map[*StringType]DataType) bool {
	if len(left) != len(right) {
		return false
	}

	found := 0
	// Both maps are iterated and for each found combination
	// of the same key/value, 'found' is incremented.
	// The maps are the same if 'found' equals the size of the
	// left map. Although it works, I'm not quite fond of this
	// solution.
	for lk, lv := range left {
		for rk, rv := range right {
			if lk.Value == rk.Value && lv.Inspect() == rv.Inspect() {
				found += 1
				continue
			}
		}
	}

	return found == len(left)
}

// Convert a StringType to ArrayType.
func (i *Interpreter) stringToArray(str *StringType) *ArrayType {
	array := &ArrayType{}
	array.Elements = []DataType{}

	for _, s := range str.Value {
		array.Elements = append(array.Elements, &StringType{Value: string(s)})
	}

	return array
}

// Convert a native Go boolean to a Boolean DataType.
func (i *Interpreter) nativeToBoolean(value bool) DataType {
	if value {
		return TRUE
	}

	return FALSE
}

// Find if a value is truthy.
func (i *Interpreter) isTruthy(object DataType) bool {
	switch object := object.(type) {
	case *BooleanType:
		return object.Value
	case *NilType:
		return false
	case *StringType:
		return object.Value != ""
	case *AtomType:
		// Atoms have always a truthy value.
		return true
	case *IntegerType:
		return object.Value != 0
	case *FloatType:
		return object.Value != 0.0
	case *ArrayType:
		return len(object.Elements) > 0
	case *DictionaryType:
		return len(object.Pairs) > 0
	default:
		return false
	}
}

func (i *Interpreter) prepareImportFilename(file string) string {
	ext := filepath.Ext(file)
	if ext == "" {
		file = file + ".ari"
	}

	return file
}

// Convert a type to an ast.Expression.
func (i *Interpreter) typeToExpression(object DataType) ast.Expression {
	switch value := object.(type) {
	case *IntegerType:
		return &ast.Integer{Value: value.Value}
	case *FloatType:
		return &ast.Float{Value: value.Value}
	case *StringType:
		return &ast.String{Value: value.Value}
	case *AtomType:
		return &ast.Atom{Value: value.Value}
	case *ArrayType:
		array := &ast.Array{}
		array.List = &ast.ExpressionList{}
		for _, v := range value.Elements {
			result := i.typeToExpression(v)
			if result == nil {
				return nil
			}
			array.List.Elements = append(array.List.Elements, result)
		}
		return array
	case *DictionaryType:
		dict := &ast.Dictionary{}
		dict.Pairs = map[ast.Expression]ast.Expression{}
		for k, v := range value.Pairs {
			key := &ast.String{Value: k.Value}
			result := i.typeToExpression(v)
			if result == nil {
				return nil
			}
			dict.Pairs[key] = result
		}
		return dict
	}

	return nil
}

// Report an error in the current location.
func (i *Interpreter) reportError(node ast.Node, message string) {
	reporter.Error(reporter.RUNTIME, node.TokenLocation(), message)
}
