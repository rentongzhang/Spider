package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	KDefaultHttpRespStatus = 1000
	KReadTimeOut           = 1001
	KBodyTooBig            = 1002
	KUpexpectedErr         = 1003
	KNewRequestErr         = 1004
	KDoRequestErr          = 1005
	KResolveTCPAddrTimeout = 1006
)

const (
	kWebUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/38.0.2125.122 Safari/537.36"
	kWapUserAgent = "Mozilla/5.0 (iPad; U; CPU OS 3_2 like Mac OS X; en-us) AppleWebKit/531.21.10 (KHTML, like Gecko) Version/4.0.4 Mobile/7B334b Safari/531.21.10"
)

type HttpUtil struct {
	client *http.Client
	ip     string
}

type HttpResponse struct {
	Status  int
	Html    string
	Header  http.Header
	Cookies []*http.Cookie
}

func NewHttpResponse() *HttpResponse {
	return &HttpResponse{
		Status:  KDefaultHttpRespStatus,
		Html:    "",
		Header:  nil,
		Cookies: nil,
	}
}

func NewHttpUtil(timeOutSeconds int) *HttpUtil {
	httpUtil := &HttpUtil{
		client: NewHttpClient(timeOutSeconds),
	}
	if httpUtil.client != nil {
		return httpUtil
	}
	return nil
}

func NewHttpUtilWithProxy(proxy string, timeOutSeconds int) *HttpUtil {
	httpUtil := &HttpUtil{
		client: NewHttpClientWithProxy(timeOutSeconds, proxy),
	}
	if httpUtil.client != nil {
		return httpUtil
	}
	return nil
}

func NewHttpUtilWithIp(ip string, timeOutSeconds int) *HttpUtil {
	httpUtil := &HttpUtil{
		client: NewHttpClientWithIp(timeOutSeconds, ip),
		ip:     ip,
	}
	if httpUtil.client != nil {
		return httpUtil
	}
	return nil
}

func (httpUtil *HttpUtil) SetCookieJar() {
	httpUtil.client = SetCookieJar(httpUtil.client)
}

func (httpUtil *HttpUtil) GetHeaderUA(needWapUserAgent bool) string {
	if needWapUserAgent {
		return kWapUserAgent
	}
	return kWebUserAgent
}

func (httpUtil *HttpUtil) isGZHtml(resp *http.Response) bool {
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		return true
	}
	return false
}

func (httpUtil *HttpUtil) needDeflate(resp *http.Response) bool {
	if strings.Contains(resp.Header.Get("Content-Encoding"), "deflate") {
		return true
	}
	return false
}

func (httpUtil *HttpUtil) GetIp() string {
	return httpUtil.ip
}

func (httpUtil *HttpUtil) setHeader(req *http.Request,
	needWapUserAgent bool) {
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	req.Header.Set("User-Agent", httpUtil.GetHeaderUA(needWapUserAgent))
}

func (httpUtil *HttpUtil) unGzipHtml(html []byte) []byte {
	var b bytes.Buffer
	b.Write(html)

	ungz, err := gzip.NewReader(&b)
	if err != nil {
		log.Println("gzip gzip.NewReader error:", err.Error(),
			" html's len:", len(html))
		return html
	}
	defer ungz.Close()

	content, err := ioutil.ReadAll(ungz)
	if err != nil {
		log.Println("gzip ioutil.ReadAll got error:", err.Error(), " html's len:",
			len(html), " content's len:", len(content))
	}

	if len(content) == 0 {
		return html
	}
	return content
}

func (httpUtil *HttpUtil) deflate(html []byte) []byte {
	// for special server data to deflate
	// see http://stackoverflow.com/questions/29513472/golang-compress-flate-module-cant-decompress-valid-deflate-compressed-http-b
	html = bytes.TrimPrefix(html, []byte("\x78\x9c"))
	reader := flate.NewReader(bytes.NewReader(html))
	defer reader.Close()
	enflated, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println("flate ioutil.ReadAll got error:", err.Error(),
			" html's len:", len(html),
			" enflated's len:", len(enflated))
	}

	if len(enflated) == 0 {
		return html
	}

	return enflated
}

func (httpUtil *HttpUtil) HttpGet(url string,
	headers map[string]string,
	cookies []*http.Cookie,
	needWapUserAgent bool,
	needDirect bool) *HttpResponse {
	ret := NewHttpResponse()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("NewRequest got error:", err.Error(), " for url:", url)
		ret.Status = KNewRequestErr
		return ret
	}

	httpUtil.setHeader(request, needWapUserAgent)
	for k, v := range headers {
		request.Header.Add(k, v)
		str := strings.ToLower(k)
		if str == "host" {
			request.Host = v
		}
	}

	if cookies != nil {
		for _, ck := range cookies {
			request.AddCookie(ck)
		}
	}

	var response *http.Response
	if needDirect {
		response, err = httpUtil.client.Do(request)
	} else {
		response, err = httpUtil.client.Transport.RoundTrip(request)
	}

	defer func() {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
	}()

	if err != nil {
		log.Println("http do req got error:", err.Error(),
			" ip:", httpUtil.ip,
			" url:", url)
		ret.Status = KDoRequestErr
		return ret
	}

	ret.Status = response.StatusCode
	ret.Cookies = response.Cookies()
	ret.Header = response.Header

	if ret.Status != 200 {
		log.Println("satus:", ret.Status, "url:", url)
		return ret
	}

	htmlByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("httpUtil.http get ioutil.ReadAll got error:",
			err.Error(),
			" read len:", len(htmlByte),
			" link:", url)
		ret.Status = KReadTimeOut
		return ret
	}

	//decompress for some site
	if httpUtil.isGZHtml(response) {
		htmlByte = httpUtil.unGzipHtml(htmlByte)
	}

	if httpUtil.needDeflate(response) {
		htmlByte = httpUtil.deflate(htmlByte)
	}

	if len(htmlByte) == 0 {
		ret.Status = KUpexpectedErr
	}

	ret.Html = string(htmlByte)
	return ret
}

func (httpUtil *HttpUtil) HttpPost(api string,
	headers map[string]string,
	postData map[string]string,
	reqCookie []*http.Cookie,
	needWapUserAgent bool) *HttpResponse {
	ret := NewHttpResponse()

	params := url.Values{}
	for key, value := range postData {
		params.Set(key, value)
	}

	postDataStr := params.Encode()
	postDataBytes := []byte(postDataStr)

	reqest, err := http.NewRequest("POST", api, bytes.NewReader(postDataBytes))
	if err != nil {
		log.Println("NewRequest got error:", err.Error(), " url:", api)
		return ret
	}

	httpUtil.setHeader(reqest, needWapUserAgent)

	reqest.Header.Set("Content-Type",
		"application/x-www-form-urlencoded; param=value")

	for k, v := range headers {
		reqest.Header.Set(k, v)
	}

	if reqCookie != nil {
		for _, ck := range reqCookie {
			reqest.AddCookie(ck)
		}
	}

	response, err := httpUtil.client.Do(reqest)

	if err != nil {
		log.Println("Client.Do got error:", err.Error())
		return ret
	}

	defer func() {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
	}()

	ret.Status = response.StatusCode
	ret.Cookies = response.Cookies()
	ret.Header = response.Header

	if response.StatusCode != 200 {
		return ret
	}

	htmlByte, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Println("httpUtil.http post ioutil.ReadAll got error:",
			err.Error())
		return ret
	}

	//decompress for some site
	if httpUtil.isGZHtml(response) {
		htmlByte = httpUtil.unGzipHtml(htmlByte)
	}

	if httpUtil.needDeflate(response) {
		htmlByte = httpUtil.deflate(htmlByte)
	}

	if len(htmlByte) == 0 {
		ret.Status = KUpexpectedErr
	}

	ret.Html = string(htmlByte)
	return ret
}
