package pipeline

type AxTableRecord struct {
	Name               string `json:"name"`
	Model              string `json:"model"`
	Layer              string `json:"layer"`
	Extends            string `json:"extends"`
	SaveDataPerCompany bool   `json:"save_data_per_company"`
	TableGroup         string `json:"table_group"`
	TableType          string `json:"table_type"`
	PrimaryIndex       string `json:"primary_index"`
	ReplacementKey     string `json:"replacement_key"`
}
