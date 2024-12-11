package example

// helperFunc retorna un valor de tipo MyType,
// esta función será una dependencia de DoStuff.
func helperFunc() MyType {
	// Usa la constante MyConst (otra dependencia)
	return MyType{Field: "Value from helperFunc, MyConst = " + string(rune(MyConst))}
}
