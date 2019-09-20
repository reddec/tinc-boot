package types

type Subnet struct {
	Subnet string `json:"subnet" form:"subnet"`
	Node   string `json:"node" form:"node"`
}
