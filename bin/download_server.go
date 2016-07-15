package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"Spider"
)

func main() {
	runtime.GOMAXPROCS(4)
	httpPort := flag.String("port", "8088", "")
	flag.Parse()

	downloadServer := spider.NewDownloadServer()

	http.HandleFunc("/download", downloadServer.HandleHttpReq)

	httpLis, err := net.Listen("tcp", ":"+*httpPort)
	if err != nil {
		log.Panicf("http listen error:%s", err.Error())
	}
	log.Println("start download server successfully!")
	http.Serve(httpLis, nil)
}
