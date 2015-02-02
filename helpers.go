package gojsonschema

func MustLoadSchema(path string) *Schema {
	s, err := NewSchema(NewReferenceLoader(path))
	if err != nil {
		panic(err)
	}
	return s
}
