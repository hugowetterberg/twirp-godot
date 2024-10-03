package twirpgodot

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

//go:embed scripts/*
var scripts embed.FS

func Generate(gen *protogen.Plugin, doc Doc) error {
	enums := map[string]Enum{}

	for _, f := range doc.Files {
		for _, e := range f.Enums {
			enums[e.FullName] = e
		}
	}

	for _, f := range doc.Files {
		for _, m := range f.Messages {
			err := messageResource(gen, m, enums)
			if err != nil {
				return fmt.Errorf("generate message resource %q: %w",
					m.FullName, err)
			}
		}

		for _, s := range f.Services {
			err := serviceNode(gen, s)
			if err != nil {
				return fmt.Errorf("generate service node %q: %w",
					s.FullName, err)
			}
		}
	}

	err := generateEnums(gen, enums)
	if err != nil {
		return fmt.Errorf("generate enums: %w", err)
	}

	scriptDir, err := scripts.ReadDir("scripts")
	if err != nil {
		return fmt.Errorf("read script directory: %w", err)
	}

	for _, file := range scriptDir {
		if !strings.HasSuffix(file.Name(), ".gd") {
			continue
		}

		path := filepath.Join("scripts", file.Name())

		f, err := scripts.Open(path)
		if err != nil {
			return fmt.Errorf("open embedded script %q: %w", path, err)
		}

		// Yes, defer in loop, but it's just a handful of files.
		defer f.Close()

		outF := gen.NewGeneratedFile(file.Name(), "")

		_, err = io.Copy(outF, f)
		if err != nil {
			return fmt.Errorf("copy embedded script %q: %w", path, err)
		}
	}

	return nil
}

func serviceNode(gen *protogen.Plugin, s Service) error {
	file := gen.NewGeneratedFile(s.FullName+".gd", "")

	buf := bufio.NewWriter(file)

	fmt.Fprintln(buf, "extends Node")
	fmt.Fprintln(buf, "")

	if s.Description != "" {
		fmt.Fprintf(buf, "# %s\n", s.Description)
	}
	fmt.Fprintf(buf, "class_name %s\n\n", godotClassName(s.FullName))

	fmt.Fprint(buf, "@export var token_source : Node\n")
	fmt.Fprint(buf, "@export var server_url : String\n\n")

	for _, m := range s.Methods {
		if m.Description != "" {
			fmt.Fprintf(buf, "# %s\n", m.Description)
		}

		fmt.Fprintf(buf, "func %s(req : %s) -> TwirpResponse:\n",
			m.Name,
			godotClassName(m.RequestType),
		)

		fmt.Fprint(buf, "\tvar treq = TwirpRequest.new()\n")
		fmt.Fprint(buf, "\ttreq.token = await token_source.get_token()\n")
		fmt.Fprint(buf, "\ttreq.server = server_url\n")
		fmt.Fprintf(buf, "\ttreq.service = %q\n", s.FullName)
		fmt.Fprintf(buf, "\ttreq.method = %q\n", m.Name)
		fmt.Fprint(buf, "\tadd_child(treq)\n")
		fmt.Fprint(buf, "\tvar resp = await treq.rpcCall(req.to_dictionary())\n")
		fmt.Fprint(buf, "\ttreq.queue_free()\n")
		fmt.Fprint(buf, "\treturn resp\n\n")
	}

	err := buf.Flush()
	if err != nil {
		return fmt.Errorf("write to file: %w", err)
	}

	return nil
}

func generateEnums(gen *protogen.Plugin, enums map[string]Enum) error {
	file := gen.NewGeneratedFile("enums.gd", "")

	buf := bufio.NewWriter(file)

	fmt.Fprintln(buf, "extends Node")
	fmt.Fprintf(buf, "class_name TwirpEnums\n\n")

	for _, e := range enums {
		fmt.Fprintf(buf, "enum %s {\n", godotClassName(e.FullName))

		for i, v := range e.Values {
			if i != 0 {
				buf.WriteString(",\n")
			}

			fmt.Fprintf(buf, "\t%s = %s", v.Name, v.Number)
		}

		fmt.Fprint(buf, "\n}\n\n")
	}

	err := buf.Flush()
	if err != nil {
		return fmt.Errorf("write to file: %w", err)
	}

	return nil
}

func messageResource(gen *protogen.Plugin, m Message, enums map[string]Enum) error {
	file := gen.NewGeneratedFile(m.FullName+".gd", "")

	buf := bufio.NewWriter(file)

	fmt.Fprintln(buf, "extends Resource")
	fmt.Fprintf(buf, "class_name %s\n", godotClassName(m.FullName))
	fmt.Fprintln(buf, "")

	for _, field := range m.Fields {
		fieldName := godotFieldName(field.Name)
		varType, comments := godotFieldType(field, enums)

		for _, c := range comments {
			fmt.Fprintln(buf, c)
		}

		fmt.Fprintf(buf, "var %s : %s\n", fieldName, varType)
	}

	fmt.Fprintln(buf, "")

	for _, field := range m.Fields {
		fieldName := godotFieldName(field.Name)

		varType, _ := godotFieldType(field, enums)
		if varType != "Dictionary" {
			continue
		}

		fmt.Fprintf(buf, "func set_%s_value(name : %s, value : %s):\n",
			fieldName, godotTypeName(field.MapKey, enums), godotTypeName(field.MapValue, enums))
		fmt.Fprintf(buf, "\tif %s == null:\n", fieldName)
		fmt.Fprintf(buf, "\t\t%s = {}\n\n", fieldName)
		fmt.Fprintf(buf, "\t%s[name] = value\n", fieldName)
		fmt.Fprint(buf, "\n")
	}

	fmt.Fprint(buf, "func to_dictionary() -> Dictionary:\n")
	fmt.Fprint(buf, "\tvar dict = {}\n\n")

	for _, field := range m.Fields {
		toDictAssignment(buf, 1, "", "", field, enums)
	}

	fmt.Fprint(buf, "\n\treturn dict\n\n")

	fmt.Fprint(buf, "static var _known_fields : Dictionary = {\n")

	for _, field := range m.Fields {
		fmt.Fprintf(buf, "\t%q: true,\n", field.Name)
	}

	fmt.Fprint(buf, "}\n\n")

	fmt.Fprintf(buf, "static func from_dictionary(dict : Dictionary, strict : bool = false) -> %s:\n",
		godotClassName(m.FullName))
	fmt.Fprintf(buf, "\tvar obj = %s.new()\n\n",
		godotClassName(m.FullName))

	for _, field := range m.Fields {
		fmt.Fprintf(buf, "\tif dict.has(%q):\n", field.Name)
		fromDictAssignment(buf, 2, "obj."+godotFieldName(field.Name), "", field, enums)
	}

	fmt.Fprint(buf, "\tif strict:\n")
	fmt.Fprint(buf, "\t\tfor key in dict:\n")
	fmt.Fprint(buf, "\t\t\tassert(_known_fields.has(key), \"ERROR: unknown field '%s'\" % key)\n")

	fmt.Fprint(buf, "\n\treturn obj\n\n")

	err := buf.Flush()
	if err != nil {
		return fmt.Errorf("write to file: %w", err)
	}

	return nil
}

func toDictAssignment(
	buf io.Writer, indent int,
	target string, source string,
	field MessageField, enums map[string]Enum,
) {
	fieldName := godotFieldName(field.Name)
	varType, _ := godotFieldType(field, enums)
	_, isEnum := enums[field.FullType]

	if target == "" {
		target = fmt.Sprintf("dict[%q]", field.Name)
	}

	if source == "" {
		source = fieldName
	}

	in := strings.Repeat("\t", indent)

	switch {
	case varType == "Dictionary":
		fmt.Fprintf(buf, "%s%s = %s\n",
			in, target, source)
	case field.IsRepeated:
		fmt.Fprintf(buf, "%sif %s.size() > 0:\n",
			in, source)
		fmt.Fprintf(buf, "%s\tvar arr_%s = []\n",
			in, source)
		fmt.Fprintf(buf, "%s\tfor v in %s:\n",
			in, source)

		item := field
		item.IsRepeated = false

		fmt.Fprintf(buf, "%s\t\tvar item\n", in)
		toDictAssignment(buf, indent+2, "item", "v", item, enums)

		fmt.Fprintf(buf, "%s\tarr_%s.append(item)\n\n",
			strings.Repeat("\t", indent+1), source)
		fmt.Fprintf(buf, "%s\t%s = arr_%s\n\n",
			in, target, source)
	case varType == "PackedByteArray":
		fmt.Fprintf(buf, "%s%s = Marshalls.raw_to_base64(%s)\n",
			in, target, source)
	case varType == "String":
		fmt.Fprintf(buf, "%sif %s != \"\":\n",
			in, source)
		fmt.Fprintf(buf, "%s\t%s = %s\n",
			in, target, source)
	case scalarTypes[varType] || isEnum:
		fmt.Fprintf(buf, "%s%s = %s\n",
			in, target, source)
	default:
		fmt.Fprintf(buf, "%sif %s != null:\n",
			in, source)
		fmt.Fprintf(buf, "%s\t%s = %s.to_dictionary()\n",
			in, target, source)
	}
}

func fromDictAssignment(
	buf io.Writer, indent int,
	target string, source string,
	field MessageField, enums map[string]Enum,
) {
	fieldName := godotFieldName(field.Name)
	varType, _ := godotFieldType(field, enums)
	_, isEnum := enums[field.FullType]

	if target == "" {
		target = fieldName
	}

	if source == "" {
		source = fmt.Sprintf("dict[%q]", field.Name)
	}

	in := strings.Repeat("\t", indent)

	fmt.Fprint(buf, in)

	switch {
	case varType == "Dictionary":
		fmt.Fprintf(buf, "%s = %s\n\n",
			target, source)
	case varType == "int":
		fmt.Fprintf(buf, "%s = int(%s)\n\n",
			target, source)
	case field.IsRepeated:
		fmt.Fprintf(buf, "for v in %s:\n",
			source)
		fmt.Fprintf(buf, "%svar item\n",
			strings.Repeat("\t", indent+1))

		item := field

		item.IsRepeated = false

		fromDictAssignment(buf, indent+1, "item", "v", item, enums)

		fmt.Fprintf(buf, "%s%s.append(item)\n\n",
			strings.Repeat("\t", indent+1), target)

	case varType == "PackedByteArray":
		fmt.Fprintf(buf, "%s = Marshalls.base64_to_raw(%s)\n",
			target, source)
	case scalarTypes[varType] || isEnum:
		fmt.Fprintf(buf, "%s = %s\n\n",
			target, source)
	default:
		fmt.Fprintf(buf, "%s = %s.from_dictionary(%s)\n",
			target, varType, source)
	}
}

func godotFieldName(name string) string {
	return "f_" + name
}

func godotFieldType(field MessageField, enums map[string]Enum) (string, []string) {
	varType := godotTypeName(field.FullType, enums)

	switch {
	case field.IsRepeated:
		return fmt.Sprintf("Array[%s]", varType), nil
	case field.IsMap:
		comment := fmt.Sprintf("# Should be typed dictionary Dictionary[%s, %s]",
			godotTypeName(field.MapKey, enums), godotTypeName(field.MapValue, enums))
		return "Dictionary", []string{comment}
	}

	return varType, nil
}

func godotTypeName(fullType string, enums map[string]Enum) string {
	if enum, ok := enums[fullType]; ok {
		return "TwirpEnums." + godotClassName(enum.FullName)
	}

	varType, isScalar := scalarTypeMap[fullType]
	if !isScalar {
		varType = godotClassName(fullType)
	}

	return varType
}

func godotClassName(fullName string) string {
	return strings.ReplaceAll(fullName, ".", "_")
}
