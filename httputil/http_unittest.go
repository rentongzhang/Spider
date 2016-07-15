package httputil

import (
	"testing"
)

func TestHttpUtil(t *testing.T) {
	client := NewHttpUtil(20)
	url := "https://www.baidu.com/"
	resp := client.HttpGet(url, nil, nil, false, false)
	if len(resp.Html) < 1000 {
		t.Error("not get expected html")
	}
	t.Log(resp.Html)
}
