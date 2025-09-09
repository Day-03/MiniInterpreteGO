package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Kind int

const (
	KindNone Kind = iota
	KindInt
	KindFloat
	KindString
	KindChar
	KindBool
	KindList
	KindFunc
)

type Value struct {
	Kind Kind
	I    int
	F    float64
	S    string
	B    bool
	L    []Value
}

func VInt(i int) Value       { return Value{Kind: KindInt, I: i} }
func VFloat(f float64) Value { return Value{Kind: KindFloat, F: f} }
func VStr(s string) Value    { return Value{Kind: KindString, S: s} }
func VChar(s string) Value { // char representado como string len=1
	return Value{Kind: KindChar, S: s}
}
func VBool(b bool) Value      { return Value{Kind: KindBool, B: b} }
func VList(xs []Value) Value  { return Value{Kind: KindList, L: xs} }
func VFunc(name string) Value { return Value{Kind: KindFunc, S: name} }
func VNone() Value            { return Value{Kind: KindNone} }

func (v Value) String() string {
	switch v.Kind {
	case KindInt:
		return fmt.Sprintf("%d", v.I)
	case KindFloat:
		return fmt.Sprintf("%g", v.F)
	case KindString, KindChar:
		return v.S
	case KindBool:
		if v.B {
			return "True"
		}
		return "False"
	case KindList:
		sb := strings.Builder{}
		sb.WriteString("[")
		for i, e := range v.L {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(e.String())
		}
		sb.WriteString("]")
		return sb.String()
	case KindFunc:
		return fmt.Sprintf("<func %s>", v.S)
	default:
		return "None"
	}
}

type Instr struct {
	Index int    // índice textual (0,1,2,...) para saltos
	Op    string // opcode
	Arg   string // parámetro crudo (puede ser vacío)
}

// Entorno de variables
type Env struct {
	Vars map[string]Value
}

// Máquina
type VM struct {
	Code     []Instr
	Index2PC map[int]int // mapa de índice->posición en slice Code
	Stack    []Value
	Env      *Env
	PC       int
}

func NewVM(code []Instr) *VM {
	m := make(map[int]int)
	for pc, ins := range code {
		m[ins.Index] = pc
	}
	return &VM{
		Code:     code,
		Index2PC: m,
		Stack:    make([]Value, 0, 64),
		Env:      &Env{Vars: map[string]Value{}},
		PC:       0,
	}
}

func (vm *VM) push(v Value) { vm.Stack = append(vm.Stack, v) }
func (vm *VM) pop() (Value, error) {
	if len(vm.Stack) == 0 {
		return Value{}, errors.New("stack underflow")
	}
	v := vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return v, nil
}

func (vm *VM) pop2() (Value, Value, error) {
	a, err := vm.pop()
	if err != nil {
		return Value{}, Value{}, err
	}
	b, err := vm.pop()
	if err != nil {
		return Value{}, Value{}, err
	}
	return a, b, nil // a=top (oper1), b=siguiente (oper2)
}

func asInt(v Value) (int, error) {
	switch v.Kind {
	case KindInt:
		return v.I, nil
	case KindFloat:
		return int(v.F), nil
	case KindBool:
		if v.B {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("se esperaba int/float/bool, got %v", v.Kind)
	}
}

func asBool(v Value) (bool, error) {
	switch v.Kind {
	case KindBool:
		return v.B, nil
	case KindInt:
		return v.I != 0, nil
	case KindFloat:
		return v.F != 0, nil
	default:
		return false, fmt.Errorf("no convertible a bool")
	}
}

func cmp(op string, a, b Value) (bool, error) {
	// Comparación numérica si ambos numéricos; si strings, compara strings; si bool, compara bool.
	isNum := func(v Value) bool { return v.Kind == KindInt || v.Kind == KindFloat || v.Kind == KindBool }
	if isNum(a) && isNum(b) {
		af, bf := float64(0), float64(0)
		switch a.Kind {
		case KindInt:
			af = float64(a.I)
		case KindFloat:
			af = a.F
		case KindBool:
			if a.B {
				af = 1
			}
		}
		switch b.Kind {
		case KindInt:
			bf = float64(b.I)
		case KindFloat:
			bf = b.F
		case KindBool:
			if b.B {
				bf = 1
			}
		}
		switch op {
		case "<":
			return af < bf, nil
		case "<=":
			return af <= bf, nil
		case ">":
			return af > bf, nil
		case ">=":
			return af >= bf, nil
		case "==":
			return af == bf, nil
		case "!=":
			return af != bf, nil
		}
	} else if a.Kind == KindString && b.Kind == KindString {
		switch op {
		case "==":
			return a.S == b.S, nil
		case "!=":
			return a.S != b.S, nil
		case "<":
			return a.S < b.S, nil
		case "<=":
			return a.S <= b.S, nil
		case ">":
			return a.S > b.S, nil
		case ">=":
			return a.S >= b.S, nil
		}
	}
	return false, fmt.Errorf("comparación no soportada para %v %s %v", a.Kind, op, b.Kind)
}

func (vm *VM) Run() error {
	for vm.PC < len(vm.Code) {
		ins := vm.Code[vm.PC]
		switch strings.ToUpper(strings.TrimSpace(ins.Op)) {

		case "LOAD_CONST":
			// soporta int, float, string ("entre comillas"), char ('c'), bool (True/False)
			arg := strings.TrimSpace(ins.Arg)
			if arg == "" {
				return fmt.Errorf("LOAD_CONST sin argumento")
			}
			// string
			if (strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"")) ||
				(strings.HasPrefix(arg, "“") && strings.HasSuffix(arg, "”")) {
				vm.push(VStr(strings.Trim(arg, "\"“”")))
			} else if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") && len([]rune(arg)) >= 3 {
				vm.push(VChar(strings.Trim(arg, "'")))
			} else if strings.EqualFold(arg, "True") || strings.EqualFold(arg, "False") {
				vm.push(VBool(strings.EqualFold(arg, "True")))
			} else if strings.Contains(arg, ".") {
				if f, err := strconv.ParseFloat(arg, 64); err == nil {
					vm.push(VFloat(f))
				} else {
					return fmt.Errorf("LOAD_CONST float inválido: %s", arg)
				}
			} else {
				if i, err := strconv.Atoi(arg); err == nil {
					vm.push(VInt(i))
				} else {
					return fmt.Errorf("LOAD_CONST int/string inválido: %s", arg)
				}
			}

		case "LOAD_FAST":
			name := strings.TrimSpace(ins.Arg)
			v, ok := vm.Env.Vars[name]
			if !ok {
				return fmt.Errorf("variable no definida: %s", name)
			}
			vm.push(v)

		case "STORE_FAST":
			name := strings.TrimSpace(ins.Arg)
			v, err := vm.pop()
			if err != nil {
				return err
			}
			vm.Env.Vars[name] = v

		case "LOAD_GLOBAL":
			// Solo funciones, p.ej. print
			name := strings.TrimSpace(ins.Arg)
			vm.push(VFunc(name))

		case "CALL_FUNCTION":
			n, err := strconv.Atoi(strings.TrimSpace(ins.Arg))
			if err != nil {
				return fmt.Errorf("CALL_FUNCTION arg inválido: %s", ins.Arg)
			}
			// Pila: [... params..., funcref]
			// Desapilar referencia a función (después de N params)
			args := make([]Value, n)
			for i := n - 1; i >= 0; i-- {
				args[i], err = vm.pop()
				if err != nil {
					return err
				}
			}
			fref, err := vm.pop()
			if err != nil {
				return err
			}
			if fref.Kind != KindFunc {
				return fmt.Errorf("CALL_FUNCTION espera referencia a función; got %v", fref.Kind)
			}
			switch strings.ToLower(fref.S) {
			case "print":
				parts := make([]string, len(args))
				for i, a := range args {
					parts[i] = a.String()
				}
				fmt.Println(strings.Join(parts, " "))
				// no empuja retorno (similar a None), si lo quieres: vm.push(VNone())
			default:
				return fmt.Errorf("función no soportada: %s", fref.S)
			}

		case "COMPARE_OP":
			op := strings.TrimSpace(ins.Arg)
			oper1, oper2, err := vm.pop2()
			if err != nil {
				return err
			}
			res, err := cmp(op, oper2, oper1) // [oper2, oper1]
			if err != nil {
				return err
			}
			vm.push(VBool(res))

		case "BINARY_ADD", "BINARY_SUBSTRACT", "BINARY_SUBTRACT",
			"BINARY_MULTIPLY", "BINARY_DIVIDE", "BINARY_MODULO":
			oper1, oper2, err := vm.pop2()
			if err != nil {
				return err
			}
			a, b := oper2, oper1 // [oper2, oper1]
			switch strings.ToUpper(strings.TrimSpace(ins.Op)) {
			case "BINARY_ADD":
				// numéricos; strings -> concatenación
				if (a.Kind == KindString || a.Kind == KindChar) && (b.Kind == KindString || b.Kind == KindChar) {
					vm.push(VStr(a.S + b.S))
				} else {
					ai, _ := asInt(a)
					bi, _ := asInt(b)
					vm.push(VInt(ai + bi))
				}
			case "BINARY_SUBSTRACT", "BINARY_SUBTRACT":
				ai, _ := asInt(a)
				bi, _ := asInt(b)
				vm.push(VInt(ai - bi))
			case "BINARY_MULTIPLY":
				ai, _ := asInt(a)
				bi, _ := asInt(b)
				vm.push(VInt(ai * bi))
			case "BINARY_DIVIDE":
				ai, _ := asInt(a)
				bi, _ := asInt(b)
				if bi == 0 {
					return fmt.Errorf("división por cero")
				}
				vm.push(VInt(ai / bi)) // entera
			case "BINARY_MODULO":
				ai, _ := asInt(a)
				bi, _ := asInt(b)
				if bi == 0 {
					return fmt.Errorf("módulo por cero")
				}
				vm.push(VInt(ai % bi))
			}

		case "BINARY_AND", "BINARY_OR":
			oper1, oper2, err := vm.pop2()
			if err != nil {
				return err
			}
			a, b := oper2, oper1
			ab, _ := asBool(a)
			bb, _ := asBool(b)
			if strings.ToUpper(ins.Op) == "BINARY_AND" {
				vm.push(VBool(ab && bb))
			} else {
				vm.push(VBool(ab || bb))
			}

		case "STORE_SUBSCR":
			// [index, array, value] -> array[index]=value
			val, arr, err := vm.pop2()
			if err != nil {
				return err
			}
			idx, err2 := vm.pop()
			if err2 != nil {
				return err2
			}
			if arr.Kind != KindList {
				return fmt.Errorf("STORE_SUBSCR espera lista")
			}
			i, err := asInt(idx)
			if err != nil || i < 0 || i >= len(arr.L) {
				return fmt.Errorf("índice inválido")
			}
			arr.L[i] = val
			// No deja nada en pila; pero debemos reflejar cambio si la lista estaba en variable.
			// Como operando venía de pila, el cambio se refleja en 'arr' local; si estaba también en variable
			// el STORE_FAST previo debió haberla actualizado. Aquí no reempujamos.

		case "BINARY_SUBSCR":
			// [index, array] -> array[index]
			idx, arr, err := vm.pop2()
			if err != nil {
				return err
			}
			if arr.Kind != KindList {
				return fmt.Errorf("BINARY_SUBSCR espera lista")
			}
			i, err := asInt(idx)
			if err != nil || i < 0 || i >= len(arr.L) {
				return fmt.Errorf("índice inválido")
			}
			vm.push(arr.L[i])

		case "BUILD_LIST":
			n, err := strconv.Atoi(strings.TrimSpace(ins.Arg))
			if err != nil {
				return fmt.Errorf("BUILD_LIST arg inválido: %s", ins.Arg)
			}
			tmp := make([]Value, n)
			for i := n - 1; i >= 0; i-- {
				v, e := vm.pop()
				if e != nil {
					return e
				}
				tmp[i] = v // preserva orden lógico [elem1 .. elemN]
			}
			vm.push(VList(tmp))

		case "JUMP_ABSOLUTE":
			target, err := strconv.Atoi(strings.TrimSpace(ins.Arg))
			if err != nil {
				return fmt.Errorf("JUMP_ABSOLUTE arg inválido")
			}
			pc, ok := vm.Index2PC[target]
			if !ok {
				return fmt.Errorf("target %d no existe", target)
			}
			vm.PC = pc
			continue

		case "JUMP_IF_TRUE", "JUMP_IF_FALSE":
			target, err := strconv.Atoi(strings.TrimSpace(ins.Arg))
			if err != nil {
				return fmt.Errorf("JUMP_IF_* arg inválido")
			}
			val, err := vm.pop()
			if err != nil {
				return err
			}
			b, err := asBool(val)
			if err != nil {
				return err
			}
			cond := (strings.ToUpper(ins.Op) == "JUMP_IF_TRUE" && b) ||
				(strings.ToUpper(ins.Op) == "JUMP_IF_FALSE" && !b)
			if cond {
				pc, ok := vm.Index2PC[target]
				if !ok {
					return fmt.Errorf("target %d no existe", target)
				}
				vm.PC = pc
				continue
			}

		case "END":
			return nil

		default:
			return fmt.Errorf("opcode no soportado: %s", ins.Op)
		}
		vm.PC++
	}
	return nil
}

// --------------------- Parser del archivo -----------------------

var opcodesWithArg = map[string]bool{
	"LOAD_CONST":    true,
	"LOAD_FAST":     true,
	"STORE_FAST":    true,
	"LOAD_GLOBAL":   true,
	"CALL_FUNCTION": true,
	"COMPARE_OP":    true,
	"JUMP_ABSOLUTE": true,
	"JUMP_IF_TRUE":  true,
	"JUMP_IF_FALSE": true,
	"BUILD_LIST":    true,
	// las demás no llevan argumento
}

func isOpcode(s string) bool {
	up := strings.ToUpper(strings.TrimSpace(s))
	if up == "" {
		return false
	}
	if up == "END" || up == "BINARY_ADD" || up == "BINARY_SUBSTRACT" || up == "BINARY_SUBTRACT" ||
		up == "BINARY_MULTIPLY" || up == "BINARY_DIVIDE" || up == "BINARY_AND" || up == "BINARY_OR" ||
		up == "BINARY_MODULO" || up == "BINARY_SUBSCR" || up == "STORE_SUBSCR" ||
		up == "LOAD_CONST" || up == "LOAD_FAST" || up == "STORE_FAST" || up == "LOAD_GLOBAL" ||
		up == "CALL_FUNCTION" || up == "COMPARE_OP" || up == "JUMP_ABSOLUTE" || up == "JUMP_IF_TRUE" ||
		up == "JUMP_IF_FALSE" || up == "BUILD_LIST" {
		return true
	}
	return false
}

func parseFile(path string) ([]Instr, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// permitir múltiples espacios/tabs: nos quedamos con la línea cruda sin tabs
		line = strings.ReplaceAll(line, "\t", " ")
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	var code []Instr
	for i := 0; i < len(lines); {
		// índice numérico
		idx, err := strconv.Atoi(lines[i])
		if err != nil {
			return nil, fmt.Errorf("se esperaba índice numérico en la línea: %q", lines[i])
		}
		i++
		if i >= len(lines) {
			return nil, fmt.Errorf("falta opcode después del índice %d", idx)
		}

		op := strings.TrimSpace(lines[i])
		i++
		arg := ""
		if opcodesWithArg[strings.ToUpper(op)] {
			if i >= len(lines) {
				return nil, fmt.Errorf("falta argumento para %s en índice %d", op, idx)
			}
			// si la “siguiente línea” parece otro opcode o un índice, aún así el formato del enunciado siempre pone el arg en la línea siguiente.
			if isOpcode(lines[i]) {
				// casos raros: si detectas aquí un opcode, probablemente el archivo no cumple el formato esperado
				return nil, fmt.Errorf("se esperaba argumento para %s en índice %d, pero vino otro opcode: %q", op, idx, lines[i])
			}
			arg = strings.TrimSpace(lines[i])
			i++
		}
		code = append(code, Instr{Index: idx, Op: op, Arg: arg})
	}
	return code, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Uso: %s <archivo.bytecode>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	code, err := parseFile(os.Args[1])
	if err != nil {
		fmt.Println("Error al parsear:", err)
		os.Exit(1)
	}
	vm := NewVM(code)
	if err := vm.Run(); err != nil {
		fmt.Println("Error en ejecución:", err)
		os.Exit(1)
	}
}
