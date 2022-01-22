package a

import "fmt"

type DB struct {
}

type foo struct {
}

func f() {
	// The pattern can be written in regular expression.
	var gopher int // want "pattern"
	print(gopher)  // want "identifier is gopher"
}

func (f foo) yey(msg string) error {
	return nil
}

func fo(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func main2() {
	fo("%d")
}
