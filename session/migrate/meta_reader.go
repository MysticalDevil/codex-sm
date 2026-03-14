package migrate

const maxSessionMetaLineBytes = 1 << 20

type metaLine struct {
	Type    string `json:"type"`
	Payload struct {
		ID  string `json:"id"`
		Cwd string `json:"cwd"`
	} `json:"payload"`
}
