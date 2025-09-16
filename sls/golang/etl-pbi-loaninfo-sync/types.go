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
