package example

// --- Requested Function ---
// DoStuff utiliza varias dependencias: llama a helperFunc, utiliza la variable MyVar y el tipo MyType
func DoStuff() {
	x := helperFunc()

	println("DoStuff called, MyVar =", MyVar)

	println("DoStuff: x.Field =", x.Field)
}
// --- Top-level Dependencies Found ---
// helperFunc retorna un valor de tipo MyType,
// esta función será una dependencia de DoStuff.
func helperFunc() MyType {

	return MyType{Field: "Value from helperFunc, MyConst = " + string(rune(MyConst))}
}
// Definimos una variable global top-level
var MyVar = "Soy una variable global"
// Definimos un tipo top-level
type MyType struct {
	Field string
}
// Definimos una constante global top-level
const MyConst = 42
