package dns_sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Fantasy135/dns-sdk/internal"
)

type CloudflareConfig struct {
	APIToken string
}

type cloudflareClient struct {
	apiToken    string
	req         *internal.Request
	api         string
	basePath    string
	contentType string
}

type zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`

	Account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"account"`

	NameServers []string `json:"name_servers"`

	Plan struct {
		LegacyID string `json:"legacy_id"`
		Name     string `json:"name"`
	} `json:"plan"`

	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on"`
}

type apiMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cloudFlareResponse[T any] struct {
	Success    bool         `json:"success"`
	Errors     []apiMessage `json:"errors"`
	Messages   []apiMessage `json:"messages"`
	Result     T            `json:"result"`
	ResultInfo *resultInfo  `json:"result_info,omitempty"`
}

type resultInfo struct {
	Count      int `json:"count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

func newCloudflareClient(cfg CloudflareConfig) (Client, error) {
	return &cloudflareClient{
		apiToken:    cfg.APIToken,
		req:         internal.Requests(),
		api:         "https://api.cloudflare.com",
		basePath:    "/client/v4",
		contentType: "application/json",
	}, nil
}

func (c cloudflareClient) DescribeUserDetail() (Accounts, error) {
	data, err := c.doRequest(http.MethodGet, c.basePath+"/accounts", nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[[]Account]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return resp.Result, nil
}

func (c cloudflareClient) DescribeDomainNameList() (Domains, error) {
	data, err := c.doRequest(http.MethodGet, c.basePath+"/zones", nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[[]zone]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	var items []Domain

	for _, z := range resp.Result {
		items = append(items, Domain{
			ID:          z.ID,
			Name:        z.Name,
			Status:      z.Status,
			NameServers: z.NameServers,
			Grade:       z.Plan.Name,
			CreatedOn:   z.CreatedOn,
			ModifiedOn:  z.ModifiedOn,
		})
	}

	return Domains(items), nil
}

func (c cloudflareClient) DescribeRecordLineList(record *Record) error {
	//TODO implement me
	panic("implement me")
}

func (c cloudflareClient) DescribeRecordList(record *Record) (DNSRecords, error) {
	params, err := extract(record, struct {
		Domain string `required:"true" json:"Domain"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest(http.MethodGet, fmt.Sprintf("%s/zones/%s/dns_records", c.basePath, zoneId), nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[[]DNSRecord]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return DNSRecords(resp.Result), nil
}

func (c cloudflareClient) DescribeSubdomainRecordList(record *Record) (DNSRecords, error) {
	params, err := extract(record, struct {
		Domain    string `required:"true" json:"Domain"`
		SubDomain string `required:"true" json:"Subdomain"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	fullName := params.Domain
	if params.SubDomain != "" && params.SubDomain != "@" {
		fullName = params.SubDomain + "." + params.Domain
	}
	data, err := c.doRequest(http.MethodGet, fmt.Sprintf("%s/zones/%s/dns_records?name=%s", c.basePath, zoneId, fullName), nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[[]DNSRecord]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return DNSRecords(resp.Result), nil
}

func (c cloudflareClient) DescribeRecord(record *Record) (*DNSRecord, error) {
	params, err := extract(record, struct {
		Domain   string `required:"true" json:"Domain"`
		RecordId string `required:"true" json:"Id"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest(http.MethodGet, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.basePath, zoneId, params.RecordId), nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[DNSRecord]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return &resp.Result, nil
}

func (c cloudflareClient) ModifyRecord(record *Record) (*DNSRecord, error) {
	params, err := extract(record, struct {
		Domain     string `required:"true" json:"Domain"`
		SubDomain  string `required:"true" json:"SubDomain"`
		RecordId   string `required:"true" json:"Id"`
		RecordType string `required:"true" json:"RecordType"`
		Value      string `required:"true" json:"Value"`
		TTL        int    `optional:"600" json:"TTL"`
		Proxied    bool   `optional:"false" json:"Proxied"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	fullName := params.Domain
	if params.SubDomain != "" && params.SubDomain != "@" {
		fullName = params.SubDomain + "." + params.Domain
	}
	payload := map[string]interface{}{
		"name":    fullName,
		"type":    params.RecordType,
		"content": params.Value,
		"ttl":     params.TTL,
		"proxied": params.Proxied,
	}
	data, err := c.doRequest(http.MethodPut, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.basePath, zoneId, params.RecordId), payload)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[DNSRecord]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return &resp.Result, nil

}

func (c cloudflareClient) CreateRecord(record *Record) (*DNSRecord, error) {
	params, err := extract(record, struct {
		Domain     string `required:"true" json:"Domain"`
		SubDomain  string `required:"true" json:"SubDomain"`
		RecordType string `required:"true" json:"RecordType"`
		Value      string `required:"true" json:"Value"`
		TTL        int    `optional:"600" json:"TTL"`
		Proxied    bool   `optional:"false" json:"Proxied"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	fullName := params.Domain
	if params.SubDomain != "" && params.SubDomain != "@" {
		fullName = params.SubDomain + "." + params.Domain
	}
	payload := map[string]interface{}{
		"name":    fullName,
		"type":    params.RecordType,
		"content": params.Value,
		"ttl":     params.TTL,
		"proxied": params.Proxied,
	}
	data, err := c.doRequest(http.MethodPost, fmt.Sprintf("%s/zones/%s/dns_records", c.basePath, zoneId), payload)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[DNSRecord]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return &resp.Result, nil
}

func (c cloudflareClient) DeleteRecord(record *Record) (*DeleteInfo, error) {
	params, err := extract(record, struct {
		Domain   string `required:"true" json:"Domain"`
		RecordId string `required:"true" json:"Id"`
	}{})
	if err != nil {
		panic(err)
	}
	zoneId, err := c.getZoneId(params.Domain)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest(http.MethodDelete, fmt.Sprintf("%s/zones/%s/dns_records/%s", c.basePath, zoneId, params.RecordId), nil)
	if err != nil {
		return nil, err
	}
	var resp cloudFlareResponse[DeleteInfo]
	err = json.Unmarshal([]byte(data), &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("cloudflare api error: %+v", resp.Errors)
	}

	return &resp.Result, nil
}

func (c cloudflareClient) getZoneId(domain string) (string, error) {
	resp, err := c.doRequest(http.MethodGet, c.basePath+"/zones?name="+domain, nil)
	if err != nil {
		return "", err
	}
	var zoneResponse struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &zoneResponse); err != nil {
		return "", err
	}
	if len(zoneResponse.Result) == 0 {
		return "", fmt.Errorf("zone not found for domain: %s", domain)
	}
	return zoneResponse.Result[0].ID, nil
}

// doRequest 统一发送请求（优先使用 requests.Requests 的 GET/POST，PUT/DELETE 走 net/http）
func (c cloudflareClient) doRequest(method, path string, payload map[string]interface{}) (string, error) {

	url := c.api + path

	c.req.ClearHeaders()
	c.req.SetHeader("Authorization", "Bearer "+c.apiToken)
	c.req.SetHeader("Content-Type", c.contentType)

	var (
		resp internal.Response
		err  error
	)

	switch method {

	case http.MethodGet:
		resp, err = c.req.Get(url)

	case http.MethodPost:
		resp, err = c.req.Post(url, payload)

	case http.MethodPut:
		resp, err = c.req.Put(url, payload)

	case http.MethodDelete:
		resp, err = c.req.Delete(url, payload)

	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(resp.Json)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
