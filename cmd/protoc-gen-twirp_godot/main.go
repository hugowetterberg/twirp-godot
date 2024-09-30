package main

import (
	"flag"
	"fmt"

	twirpgodot "github.com/hugowetterberg/twirp-godot"
	"google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	var flags flag.FlagSet

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = internal_gengo.SupportedFeatures

		doc := twirpgodot.StructureDump(gen)

		err := twirpgodot.Generate(gen, doc)
		if err != nil {
			return fmt.Errorf("generate: %w", err)
		}

		return nil
	})
}
