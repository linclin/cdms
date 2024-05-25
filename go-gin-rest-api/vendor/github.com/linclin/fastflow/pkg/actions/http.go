package actions

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/linclin/fastflow/pkg/entity/run"
)

const (
	ActionHTTP = "http"
)

type HTTPParams struct {
	Method    string            `yaml:"method" json:"method"`
	URL       string            `yaml:"url" json:"url"`
	RawBody   string            `yaml:"rawBody" json:"rawBody"`
	Header    map[string]string `yaml:"header" json:"header"`
	BasicAuth map[string]string `yaml:"basicauth" json:"basicauth"`
	AuthToken string            `yaml:"authtoken" json:"authtoken"`
}
type HTTP struct {
}

// Name Action name
func (h *HTTP) Name() string {
	return ActionHTTP
}

// ParameterNew
func (h *HTTP) ParameterNew() interface{} {
	return &HTTPParams{}
}

// Run
func (h *HTTP) Run(ctx run.ExecuteContext, params interface{}) (err error) {
	ctx.Trace("[Action http]start " + fmt.Sprintln("params", params))
	p, ok := params.(*HTTPParams)
	if !ok {
		err = fmt.Errorf("[Action http]params type mismatch, want *HTTPParams, got %T", params)
		ctx.Trace(err.Error())
		return err
	}
	if p.URL == "" || p.Method == "" {
		err = fmt.Errorf("[Action http]HTTPParams method/url cannot be empty")
		ctx.Trace(err.Error())
		return err
	}
	resp := &resty.Response{}
	httpClient := resty.New().R().
		SetHeader("accept", "application/json").
		SetHeader("Content-Type", "application/json")

	if len(p.Header) > 0 {
		for k, v := range p.Header {
			httpClient.SetHeader(k, v)
		}
	}
	if p.AuthToken != "" {
		httpClient.SetAuthToken(p.AuthToken)
	}
	if len(p.BasicAuth) > 0 {
		httpClient.SetBasicAuth(p.BasicAuth["username"], p.BasicAuth["password"])
	}
	switch strings.ToUpper(p.Method) {
	case "POST":
		resp, err = httpClient.SetBody(p.RawBody).Post(p.URL)
	case "GET":
		resp, err = httpClient.Get(p.URL)
	case "PUT":
		resp, err = httpClient.SetBody(p.RawBody).Put(p.URL)
	case "DELETE":
		resp, err = httpClient.SetBody(p.RawBody).Delete(p.URL)
	default:
		err := fmt.Errorf("[Action http]HTTPParams method %s not support", strings.ToUpper(p.Method))
		ctx.Trace(err.Error())
		return err
	}
	if err != nil {
		err = fmt.Errorf("[Action http]http request failed, %w", err)
		ctx.Trace(err.Error())
		return err
	}
	if resp.StatusCode() != 200 {
		err = fmt.Errorf("[Action http]http request failed,StatusCode %d,Resp %s", resp.StatusCode(), string(resp.Body()))
		ctx.Trace(err.Error())
		return err
	}
	ctx.WithValue(p.Method+p.URL, string(resp.Body()))
	ctx.Trace("[Action http]success " + fmt.Sprintln("params", params))
	return nil
}
