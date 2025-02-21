package utils

// Shared struct
type ExportRequest struct {
	Query      string `json:"query"`
	OutputFile string `json:"output_file"`
}
