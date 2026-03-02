package dns_sdk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Fantasy135/dns-sdk/internal"
)

type TencentConfig struct {
	SecretID  string
	SecretKey string
}

type tencentUserInfo struct {
	Email     string `json:"Email"`
	ID        int    `json:"Id"`
	Uin       int    `json:"Uin"`
	Name      string `json:"Name"`
	UserGrade string `json:"UserGrade"`
}

type tencentError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type tencentDomain struct {
	DomainId     int      `json:"DomainId"`
	Name         string   `json:"Name"`
	Status       string   `json:"Status"`
	Grade        string   `json:"Grade"`
	EffectiveDNS []string `json:"EffectiveDNS"`
	CreatedOn    string   `json:"CreatedOn"`
	UpdatedOn    string   `json:"UpdatedOn"`
}

type tencentRecordLite struct {
	RecordId int    `json:"RecordId"`
	Name     string `json:"Name"`
	Value    string `json:"Value"`
	Type     string `json:"Type"`
	TTL      int    `json:"TTL"`
	Line     string `json:"Line"`
}

type tencentRecord struct {
	Id         int    `json:"Id"`
	SubDomain  string `json:"SubDomain"`
	RecordType string `json:"RecordType"`
	RecordLine string `json:"RecordLine"`
	Value      string `json:"Value"`
	TTL        int    `json:"TTL"`
}

type tencentResponse struct {
	Response struct {
		UserInfo   tencentUserInfo `json:"UserInfo"`
		DomainList []tencentDomain
		RecordList []tencentRecordLite `json:"RecordList"`
		RecordInfo tencentRecord       `json:"RecordInfo"`
		Error      *tencentError       `json:"Error,omitempty"`
		RecordId   int                 `json:"RecordId"`
		RequestId  string              `json:"RequestId"`
	} `json:"Response"`
}

type tencentClient struct {
	secretId    string
	secretKey   string
	service     string
	req         *internal.Request
	api         string
	version     string
	algorithm   string
	contentType string
	httpMethod  string
}

func newTencentClientClient(cfg TencentConfig) (Client, error) {
	return &tencentClient{
		secretId:    cfg.SecretID,
		secretKey:   cfg.SecretKey,
		req:         internal.Requests(),
		service:     "dnspod",
		api:         "dnspod.tencentcloudapi.com",
		version:     "2021-03-23",
		algorithm:   "TC3-HMAC-SHA256",
		contentType: "application/json",
		httpMethod:  "POST",
	}, nil
}

func (t tencentClient) DescribeUserDetail() (string, error) {
	data, err := t.doRequest("DescribeUserDetail", nil)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	acc := account{
		ID:       resp.Response.UserInfo.ID,
		Name:     resp.Response.UserInfo.Uin,
		UserType: resp.Response.UserInfo.UserGrade,
	}

	out, err := json.MarshalIndent([]account{acc}, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) DescribeDomainNameList() (string, error) {
	data, err := t.doRequest("DescribeDomainList", nil)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	var domains []domain

	for _, d := range resp.Response.DomainList {
		domains = append(domains, domain{
			ID:          strconv.Itoa(d.DomainId),
			Name:        d.Name,
			Status:      d.Status,
			NameServers: d.EffectiveDNS,
			Grade:       d.Grade,
			CreatedOn:   d.CreatedOn,
			ModifiedOn:  d.UpdatedOn,
		})
	}

	out, err := json.MarshalIndent(domains, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) DescribeRecordLineList(record *Record) error {
	//TODO implement me
	panic("implement me")
}

func (t tencentClient) DescribeRecordList(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain string `required:"true" json:"Domain"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("DescribeRecordList", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	var records []dnsRecord

	for _, d := range resp.Response.RecordList {
		records = append(records, dnsRecord{
			ID:      strconv.Itoa(d.RecordId),
			Name:    d.Name,
			Type:    d.Type,
			Content: d.Value,
			TTL:     d.TTL,
			Line:    d.Line,
		})
	}

	out, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) DescribeSubdomainRecordList(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain    string `required:"true" json:"Domain"`
		SubDomain string `required:"true" json:"Subdomain"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("DescribeRecordList", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	var records []dnsRecord

	for _, d := range resp.Response.RecordList {
		records = append(records, dnsRecord{
			ID:      strconv.Itoa(d.RecordId),
			Name:    d.Name,
			Type:    d.Type,
			Content: d.Value,
			TTL:     d.TTL,
			Line:    d.Line,
		})
	}

	out, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) DescribeRecord(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain string `required:"true" json:"Domain"`
		Id     int    `required:"true" json:"RecordId"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("DescribeRecord", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	recording := dnsRecord{
		ID:      strconv.Itoa(resp.Response.RecordInfo.Id),
		Name:    resp.Response.RecordInfo.SubDomain + "." + params.Domain,
		Type:    resp.Response.RecordInfo.RecordType,
		Content: resp.Response.RecordInfo.Value,
		TTL:     resp.Response.RecordInfo.TTL,
		Line:    resp.Response.RecordInfo.RecordLine,
	}

	out, err := json.MarshalIndent(recording, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) ModifyRecord(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain     string `required:"true" json:"Domain"`
		SubDomain  string `optional:"@" json:"SubDomain"`
		RecordId   int    `required:"true" json:"RecordId"`
		RecordType string `required:"true" json:"RecordType"`
		RecordLine string `optional:"默认" json:"RecordLine"`
		Value      string `required:"true" json:"Value"`
		TTL        int    `optional:"600" json:"TTL"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("ModifyRecord", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	recording := dnsRecord{
		ID:      strconv.Itoa(resp.Response.RecordId),
		Name:    params.SubDomain + "." + params.Domain,
		Type:    params.RecordType,
		Content: params.Value,
		TTL:     params.TTL,
		Line:    params.RecordLine,
	}

	out, err := json.MarshalIndent(recording, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil

}

func (t tencentClient) CreateRecord(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain     string `required:"true" json:"Domain"`
		SubDomain  string `optional:"@" json:"SubDomain"`
		RecordType string `required:"true" json:"RecordType"`
		RecordLine string `optional:"默认" json:"RecordLine"`
		Value      string `required:"true" json:"Value"`
		TTL        int    `optional:"600" json:"TTL"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("CreateRecord", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	recording := dnsRecord{
		ID:      strconv.Itoa(resp.Response.RecordId),
		Name:    params.SubDomain + "." + params.Domain,
		Type:    params.RecordType,
		Content: params.Value,
		TTL:     params.TTL,
		Line:    params.RecordLine,
	}

	out, err := json.MarshalIndent(recording, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) DeleteRecord(record *Record) (string, error) {
	params, err := extract(record, struct {
		Domain   string `required:"true" json:"Domain"`
		RecordId int    `required:"true" json:"RecordId"`
	}{})
	if err != nil {
		panic(err)
	}

	data, err := t.doRequest("DescribeRecord", params)
	if err != nil {
		return "", err
	}

	var resp tencentResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", err
	}

	if resp.Response.Error != nil {
		return "", fmt.Errorf("tencent api error: %+v", resp.Response.Error)
	}

	deleterious := deleteInfo{
		ID: resp.Response.RequestId,
	}

	out, err := json.MarshalIndent(deleterious, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (t tencentClient) doRequest(action string, params any) (string, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	// 1) 构建请求载荷（JSON）并保留字节以确保与发送一致
	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal params to json: %w", err)
	}
	payload := string(payloadBytes)

	signedHeaders := "content-type;host;x-tc-action"

	// 2) 规范请求 CanonicalRequest
	canonicalRequest := t.CanonicalRequest([]byte(payload), signedHeaders, action)

	// 3) 计算签名 Signature
	credentialScope, timestamp, signature := t.signature(canonicalRequest)

	// 5) 构建 Authorization 头
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.algorithm,
		t.secretId,
		credentialScope,
		signedHeaders,
		signature,
	)

	// 6) 使用自定义 requests.Request 发送 HTTP 请求
	t.req.ClearHeaders()
	t.req.SetHeader("Authorization", authorization)
	t.req.SetHeader("Content-Type", t.contentType)
	t.req.SetHeader("Host", t.api)
	t.req.SetHeader("X-TC-Action", action)
	t.req.SetHeader("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	t.req.SetHeader("X-TC-Version", t.version)

	resp, err := t.req.Post("https://"+t.api, payloadBytes)
	if err != nil {
		return "", fmt.Errorf("failed to send http request: %w", err)
	}
	return t.formatJSONUtf(resp.Text), nil
}

func (t tencentClient) CanonicalRequest(payload []byte, signedHeaders, action string) string {
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n", t.contentType, t.api, strings.ToLower(action))
	hashedRequestPayload := fmt.Sprintf("%x", sha256.Sum256(payload))
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		t.httpMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload,
	)
}

func (t tencentClient) signature(canonicalRequest string) (string, int64, string) {
	// 3) 构建待签字符串 StringToSign
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, t.service)
	hashedCanonicalRequest := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s", t.algorithm, timestamp, credentialScope, hashedCanonicalRequest)

	// 4) 计算签名 Signature
	hmacSHA256 := func(msg string, key []byte) []byte {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(msg))
		return mac.Sum(nil)
	}
	secretDate := hmacSHA256(date, []byte("TC3"+t.secretKey))
	secretService := hmacSHA256(t.service, secretDate)
	secretSigning := hmacSHA256("tc3_request", secretService)
	return credentialScope, timestamp, hex.EncodeToString(hmacSHA256(stringToSign, secretSigning))
}

func (t tencentClient) formatJSONUtf(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return s
	}
	return buf.String()
}
