package httputil

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

func noRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("don't redirect")
}

func NewHttpClient(timeOutSeconds int) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				timeout := time.Duration(timeOutSeconds) * time.Second
				deadline := time.Now().Add(timeout)
				c, err := net.DialTimeout(netw, addr, timeout)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeOutSeconds) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func NewNoRedirectHttpClient(timeOutSeconds int) *http.Client {
	ret := NewHttpClient(timeOutSeconds)
	ret.CheckRedirect = noRedirect
	return ret
}

func SetCookieJar(client *http.Client) *http.Client {
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Panicf(err.Error())
	}
	client.Jar = jar
	return client
}

func NewNoRedirectHttpClientWithIp(timeOutSeconds int,
	ip string) *http.Client {
	ret := NewHttpClientWithIp(timeOutSeconds, ip)
	ret.CheckRedirect = noRedirect
	return ret
}

func NewHttpClientWithIp(timeOutSeconds int, ip string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				lAddr, err := net.ResolveTCPAddr(netw, ip+":0")

				if err != nil {
					return nil, err
				}
				//for local ip and remote ip debug
				log.Println("get local_ip:", lAddr.IP.String())

				rAddr, err := net.ResolveTCPAddr(netw, addr)
				if err != nil {
					return nil, err
				}
				//for local ip and remote ip debug
				log.Println("local_ip:", lAddr.IP.String(), " remote_ip:", rAddr.IP.String())

				conn, err := net.DialTCP(netw, lAddr, rAddr)
				if err != nil {
					return nil, err
				}
				deadline := time.Now().Add(time.Duration(timeOutSeconds) * time.Second)
				conn.SetDeadline(deadline)
				return conn, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeOutSeconds) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func NewNoRedirectHttpClientWithProxy(timeOutSeconds int,
	proxy string) *http.Client {
	ret := NewHttpClientWithProxy(timeOutSeconds, proxy)
	ret.CheckRedirect = noRedirect
	return ret
}

func NewHttpClientWithProxy(timeOutSeconds int, proxy string) *http.Client {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		log.Println("error, got not valid proxy:", proxy, " get err:", err.Error())
		return nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				timeout := time.Duration(timeOutSeconds) * time.Second
				deadline := time.Now().Add(timeout)
				c, err := net.DialTimeout(netw, addr, timeout)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeOutSeconds) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			Proxy:                 http.ProxyURL(proxyUrl),
		},
	}
}

func GetLocalIp(netInterface, prefix string) ([]string, error) {
	ret := []string{}
	ief, err := net.InterfaceByName(netInterface)
	if err != nil {
		return ret, err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return ret, err
	}

	for _, addr := range addrs {
		ip := addr.(*net.IPNet).IP.String()
		if strings.HasPrefix(ip, prefix) {
			ret = append(ret, ip)
		}
	}
	return ret, nil
}
