package main

import (
	"errors"

	"github.com/extism/go-pdk"
)

type Input struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Diffs       []byte            `json:"diffs"`
	Vars        map[string]string `json:"vars"`
}

type Output struct {
	Comment string `json:"comment"`
}

//go:wasmexport hello
func Hello() int32 {
	input := Input{}
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	name, ok := input.Vars["demo_name"]
	if !ok {
		pdk.SetError(errors.New("DEMO_NAME is not provided"))
		return 1
	}

	output := Output{
		Comment: "hello " + input.Author + " from " + name,
	}

	pdk.OutputJSON(output)

	return 0
}

func main() {}
