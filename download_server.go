package spider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"Spider/httputil"
)

type DownloadServer struct {
	httpClient *httputil.HttpUtil
}

func NewDownloadServer() *DownloadServer {
	return &DownloadServer{
		httpClient: httputil.NewHttpUtil(30),
	}
}

func (self *DownloadServer) HandleHttpReq(w http.ResponseWriter,
	r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			statckBuf := make([]byte, 64*1024)
			runtime.Stack(statckBuf, false)
			log.Printf("downloader recover:%v, stack%s",
				r, string(statckBuf))
		}
	}()

	if r.Method == "POST" {
		log.Println("not support post method for:", r.RequestURI)
		fmt.Fprint(w, "not support post method")
		return
	}

	url := r.URL.Query().Get("url")
	resp := self.httpClient.HttpGet(url, nil, nil, false, false)
	data, _ := json.Marshal(resp)
	fmt.Fprint(w, string(data))
}
