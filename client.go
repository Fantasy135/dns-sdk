package dns_sdk

import (
	"fmt"
	"reflect"
	"strconv"
)

type account struct {
	ID       any    `json:"id"`
	Name     any    `json:"name"`
	UserType string `json:"type"`
}

type domain struct {
	ID          string
	Name        string
	Status      string
	NameServers []string
	Grade       string
	CreatedOn   string
	ModifiedOn  string
}

type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Line    string `json:"Line"`
	Proxied bool   `json:"proxied"`
}

type deleteInfo struct {
	ID string `json:"id"`
}

type Client interface {
	DescribeUserDetail() (string, error)
	DescribeDomainNameList() (string, error)
	DescribeRecordLineList(*Record) error
	DescribeRecordList(*Record) (string, error)
	DescribeSubdomainRecordList(*Record) (string, error)
	DescribeRecord(*Record) (string, error)
	CreateRecord(*Record) (string, error)
	ModifyRecord(*Record) (string, error)
	DeleteRecord(*Record) (string, error)
}

func NewClient(cfg any) (Client, error) {
	switch c := cfg.(type) {
	case CloudflareConfig:
		return newCloudflareClient(c)

	case *CloudflareConfig:
		return newCloudflareClient(*c)

	case TencentConfig:
		return newTencentClientClient(c)

	case *TencentConfig:
		return newTencentClientClient(*c)

	default:
		return nil, fmt.Errorf("unsupported DNS config type: %T", cfg)
	}
}

type Record struct {
	Domain      string
	SubDomain   string
	RecordType  string
	RecordLine  string
	TTL         int
	Value       string
	RecordId    any
	Weight      string
	DomainGrade string
	Proxied     bool
}

func extract[B any, P any](b *B, _ P) (*P, error) {
	if b == nil {
		return nil, fmt.Errorf("record is nil")
	}

	bv := reflect.ValueOf(b).Elem()
	bt := bv.Type()

	pt := reflect.TypeOf((*P)(nil)).Elem()
	pv := reflect.New(pt).Elem()

	for i := 0; i < pt.NumField(); i++ {
		pf := pt.Field(i)
		name := pf.Name

		field := pv.Field(i)
		target := pf.Type

		// ===== 找 builder 字段 =====
		var (
			val      any
			hasValue bool
		)

		if bf, ok := bt.FieldByName(name); ok {
			bvf := bv.FieldByIndex(bf.Index)
			if bvf.IsValid() && !bvf.IsZero() {
				val = bvf.Interface()
				hasValue = true
			}
		}

		// ===== required 校验 =====
		if pf.Tag.Get("required") == "true" && !hasValue {
			return nil, fmt.Errorf("%s is required", name)
		}

		// ===== optional 默认值 =====
		if !hasValue {
			if def := pf.Tag.Get("optional"); def != "" {
				val = def
				hasValue = true
			}
		}

		if !hasValue {
			continue
		}

		// ===== 类型规范化 =====
		switch target.Kind() {

		case reflect.String:
			switch v := val.(type) {
			case string:
				field.SetString(v)
			case int:
				field.SetString(strconv.Itoa(v))
			case int64:
				field.SetString(strconv.FormatInt(v, 10))
			case uint64:
				field.SetString(strconv.FormatUint(v, 10))
			default:
				return nil, fmt.Errorf("%s: cannot convert %T to string", name, val)
			}

		case reflect.Int, reflect.Int64:
			switch v := val.(type) {
			case int:
				field.SetInt(int64(v))
			case int64:
				field.SetInt(v)
			case string:
				n, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("%s: invalid int string", name)
				}
				field.SetInt(n)
			default:
				return nil, fmt.Errorf("%s: cannot convert %T to int", name, val)
			}

		case reflect.Bool:
			switch v := val.(type) {
			case bool:
				field.SetBool(v)
			case string:
				b, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("%s: invalid bool string", name)
				}
				field.SetBool(b)
			default:
				return nil, fmt.Errorf("%s: cannot convert %T to bool", name, val)
			}

		default:
			rv := reflect.ValueOf(val)
			if rv.Type().AssignableTo(target) {
				field.Set(rv)
			} else {
				return nil, fmt.Errorf("%s: type mismatch %T -> %s", name, val, target)
			}
		}
	}

	out := pv.Interface().(P)
	return &out, nil
}
