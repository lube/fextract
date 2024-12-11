package example

// DoStuff utiliza varias dependencias: llama a helperFunc, utiliza la variable MyVar y el tipo MyType
func DoStuff() {
	x := helperFunc()
	// x es del tipo MyType, retornado por helperFunc

	// Usamos una variable global MyVar
	println("DoStuff called, MyVar =", MyVar)

	// Imprimimos un campo de x (MyType)
	println("DoStuff: x.Field =", x.Field)
}
