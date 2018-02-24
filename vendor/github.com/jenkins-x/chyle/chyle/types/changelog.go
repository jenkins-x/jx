package types

// Changelog represents a changelog entry with datas extracted and metadatas
type Changelog struct {
	Datas     []map[string]interface{} `json:"datas"`
	Metadatas map[string]interface{}   `json:"metadatas"`
}
