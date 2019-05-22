package swagger

type PropertyType string
type PropertyFormat string

const (
	FormatDouble   PropertyFormat = "double"
	FormatInt32    PropertyFormat = "int32"
	FormatInt64    PropertyFormat = "int64"
	FormatDateTime PropertyFormat = "date-time"
)

const (
	TypeString  PropertyType = "string"
	TypeInteger PropertyType = "integer"
	TypeNumber  PropertyType = "number"
	TypeArray   PropertyType = "array"
	TypeBoolean PropertyType = "boolean"
)

type PropertyItems struct {
	Ref    string         `json:"$ref"`
	Format PropertyFormat `json:"format"`
	Type   PropertyType   `json:"type"`
}

type Property struct {
	Ref    string         `json:"$ref"`
	Format PropertyFormat `json:"format"`
	Type   PropertyType   `json:"type"`
	Items  *Property      `json:"items"`
}

type Properties map[string]*Property

type Definition struct {
	Required   []string   `json:"required"`
	Type       string     `json:"type"`
	Properties Properties `json:"properties"`
}

type Definitions map[string]*Definition

type Spec struct {
	Definitions Definitions `json:"definitions"`
}
