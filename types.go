package twirpgodot

var scalarTypeMap = map[string]string{
	"bool":   "bool",
	"int32":  "int",
	"int64":  "int",
	"float":  "float",
	"double": "float",
	"string": "String",
	"bytes":  "PackedByteArray",
}

var scalarTypes = map[string]bool{}

var reservedWords = map[string]bool{
	"func":  true,
	"if":    true,
	"match": true,
	"range": true,
	"name":  true,
	"value": true,
}

func init() {
	for _, t := range scalarTypeMap {
		reservedWords[t] = true
		scalarTypes[t] = true
	}
}
