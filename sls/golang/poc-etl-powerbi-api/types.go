package main

type apiResponse struct {
	Results []result `json:"results"`
}

type result struct {
	Tables []table `json:"tables"`
}

type table struct {
	Rows []map[string]any `json:"rows"`
}

type investment struct {
	InvestmentName         string
	PeriodToDate           float64
	PeriodToDatePresentage float64
	YearToDate             float64
	YearToDatePresentage   float64
}

type powerbiApiSecret struct {
	TenantId     string `json:"tenantId"`
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type secretResult struct {
	AccessToken string `json:"access_token"`
}

type powerbiApplicationCfg struct {
	WorkspaceID string
	DatasetID   string
}
