package restful

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"resty.dev/v3"
)

type Response struct {
	Data interface{} `json:"data"`
}

type ErrorResponse struct {
	Error map[string]interface{} `json:"error"`
}

type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e ErrorData) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}

func NewErrorData(code int, msg string) ErrorData {
	return ErrorData{Code: code, Message: msg}
}

func NewErrorResponse(err error) ErrorResponse {
	e := ErrorResponse{}
	e.Error = make(map[string]interface{})

	switch v := err.(type) {
	case ErrorData:
		e.Error["code"] = v.Code
		e.Error["message"] = v.Message
	default:
		e.Error["code"] = -1
		e.Error["message"] = v.Error()
	}
	return e
}

func NewResponse() Response {
	r := Response{}
	r.Data = make(map[string]interface{})
	return r
}
func NewResponseWith(data interface{}) *Response {
	return &Response{
		Data: data,
	}
}

func E(code int, msg string) ErrorResponse {
	return NewErrorResponse(ErrorData{
		Code:    code,
		Message: msg,
	})
}

// 发送失败响应体
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(code, E(code, msg))
}

// 发送成功响应体
func Ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Data: data})
}

func (r Response) Set(key string, value interface{}) Response {
	m, ok := r.Data.(map[string]interface{})
	if !ok {
		return r
	}
	m[key] = value
	return r
}

func interfaceToURLValues(data interface{}) (url.Values, error) {
	if data == nil {
		return nil, nil
	}
	values := url.Values{}

	// 确保输入是一个 map 类型
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("expected a map, got %T", data)
	}

	// 遍历 map，将 key 和 value 转为 url.Values 格式
	for _, key := range v.MapKeys() {
		strKey := fmt.Sprintf("%v", key) // 将 key 转为字符串
		val := v.MapIndex(key)

		// 处理 value 为切片的情况
		if val.Kind() == reflect.Slice {
			for i := 0; i < val.Len(); i++ {
				values.Add(strKey, fmt.Sprintf("%v", val.Index(i)))
			}
		} else {
			// 单值情况
			values.Add(strKey, fmt.Sprintf("%v", val))
		}
	}

	return values, nil
}

func parseResponse[T any](bytes []byte) (result *T, err error) {
	// log.Debug().Msg("response: " + string(bytes))
	result = new(T)
	type Body[T any] struct {
		Data  *T         `json:"data"`
		Error *ErrorData `json:"error"`
	}
	body := Body[T]{}
	err = json.Unmarshal(bytes, &body)
	if err != nil {
		// log.Debug().Err(err).Msgf("invalid json response: %s", string(bytes))
		fmt.Printf("invalid json response: %s\n", string(bytes))
		return nil, NewErrorData(-1, "invalid response format: "+string(bytes))
	}
	if body.Error != nil {
		return nil, *body.Error
	}
	if body.Data == nil {
		return nil, NewErrorData(-1, "data is nil")
	}
	return body.Data, nil
}

func Post[T any](instance *Restful, url string, data any) (result *T, err error) {
	instance.Resty.SetHeader("Content-Type", "application/json")

	response, err := instance.Resty.R().
		SetBody(data). // default request content type is JSON
		Post(url)

	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !instance.Envolope {
		body := new(T)
		err := json.Unmarshal(response.Bytes(), &body)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return body, nil
	}
	return parseResponse[T](response.Bytes())
}

func Get[T any](instance *Restful, url string, data any) (result *T, err error) {
	values, err := interfaceToURLValues(data)
	if err != nil {
		return nil, err
	}

	response, err := instance.Resty.R().
		SetQueryParamsFromValues(values).
		Get(url)
	if err != nil {
		return nil, NewErrorData(-1, err.Error())
	}
	return parseResponse[T](response.Bytes())
}

func Request[ResponseType any](rest *Restful, doRequest func(*Restful) (*resty.Response, error)) (*ResponseType, error) {
	if doRequest == nil {
		return nil, errors.New("doRequest is nil")
	}
	response, err := doRequest(rest)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return parseResponse[ResponseType](response.Bytes())
}

type Restful struct {
	Resty    *resty.Client
	Envolope bool //是否是信封的应答格式
}

func (the *Restful) SetHeader(key, value string) {
	the.Resty.SetHeader(key, value)
}

func (the *Restful) SetBearer(token string) {
	the.Resty.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

// GetRequestBaseURL 返回完整的请求基础地址（协议+域名）
func GetBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// 优先使用 X-Forwarded-* 头（代理环境）
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func New(baseurl string) (api *Restful) {
	instance := resty.New()
	instance.SetBaseURL(baseurl)
	instance.SetTimeout(time.Second * 15)
	return &Restful{Resty: instance, Envolope: true}
}

type ProgressCallback func(writer *Writer) error

type Writer struct {
	Total      int64
	Downloaded int64
	StartTime  time.Time
	Speed      float64 //  byte/s
	Progress   ProgressCallback
}

func (wc *Writer) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += int64(n)
	duration := time.Since(wc.StartTime).Seconds()
	wc.Speed = float64(wc.Downloaded) / duration

	return n, wc.Progress(wc)
}

type Config struct {
	URL      string
	SaveAs   string
	Progress ProgressCallback
	Timeout  time.Duration
}

func Download(config *Config) error {
	client := resty.New()
	defer client.Close()
	resp, err := client.R().SetTimeout(config.Timeout).
		// SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7").
		SetHeader("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Mobile Safari/537.36 Edg/137.0.0.0").
		SetDoNotParseResponse(true).
		Get(config.URL)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode() != 200 {
		return errors.Errorf("%s %s error: %s", resp.Request.Method, resp.Request.URL, resp.Status())
	}

	contentLength, _ := strconv.Atoi(resp.Header().Get("Content-Length"))

	counter := &Writer{
		Total:     int64(contentLength),
		StartTime: time.Now(),
		Progress:  config.Progress,
	}

	tmpFile := config.SaveAs + ".downloading"
	out, err := os.Create(tmpFile)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		out.Close()
		os.RemoveAll(tmpFile)
	}()

	_, err = io.Copy(out, io.TeeReader(resp.RawResponse.Body, counter))
	if err != nil {
		return errors.WithStack(err)
	}

	out.Close()
	if contentLength <= 0 {
		os.Rename(tmpFile, config.SaveAs)
		return nil
	}
	fileInfo, err := os.Stat(tmpFile)
	if err != nil {
		return errors.WithStack(err)
	}
	if fileInfo.Size() != int64(contentLength) {
		return errors.New("download error, incomplete file")
	}
	os.Rename(tmpFile, config.SaveAs)
	return nil
}
