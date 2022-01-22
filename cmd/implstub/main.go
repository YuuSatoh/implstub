package main

import (
	"implstub"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(implstub.ImplStubAnalyzer)
}
