package main

import (
	"errors"

	"github.com/extism/go-pdk"
)

type PluginInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Diffs       []byte `json:"diffs"`
}

type PluginOutput struct {
	Comment string `json:"comment"`
}

//go:wasmexport hello
func Hello() int32 {
	input := PluginInput{}
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	name, ok := pdk.GetConfig("demo_name")
	if !ok {
		pdk.SetError(errors.New("DEMO_NAME is not provided"))
		return 1
	}

	output := PluginOutput{
		Comment: "hello " + input.Author + " from " + name,
	}

	pdk.OutputJSON(output)

	return 0
}

func main() {}
