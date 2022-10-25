package utils

import (
	"crypto/tls"
	"fmt"
	"github.com/labring/lvscare/internal/glog"
	"github.com/labring/lvscare/internal/ipvs"
	"net"
	"net/http"
	"strconv"
)

//SplitServer is
func SplitServer(server string) (string, uint16) {
	glog.V(8).Infof("server %s", server)

	ip, port, err := net.SplitHostPort(server)
	if err != nil {
		glog.Errorf("SplitServer error: %v.", err)
		return "", 0
	}
	glog.V(8).Infof("SplitServer debug: IP: %s, Port: %s", ip, port)
	p, err := strconv.Atoi(port)
	if err != nil {
		glog.Warningf("SplitServer error: %v", err)
		return "", 0
	}
	return ip, uint16(p)
}

//IsHTTPAPIHealth is check http error
func IsHTTPAPIHealth(ip, port, path, schem string) bool {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	url := fmt.Sprintf("%s://%s:%s%s", schem, ip, port, path)
	resp, err := http.Get(url)
	if err != nil {
		glog.V(8).Infof("IsHTTPAPIHealth error: %v", err)
		return false
	}
	defer resp.Body.Close()

	_ = resp
	return true
}

func BuildVirtualServer(vip string) *ipvs.VirtualServer {
	ip, port := SplitServer(vip)
	virServer := &ipvs.VirtualServer{
		Address:   net.ParseIP(ip),
		Protocol:  "TCP",
		Port:      port,
		Scheduler: "rr",
		Flags:     0,
		Timeout:   0,
	}
	return virServer
}

func BuildRealServer(real string) *ipvs.RealServer {
	ip, port := SplitServer(real)
	realServer := &ipvs.RealServer{
		Address: net.ParseIP(ip),
		Port:    port,
		Weight:  1,
	}
	return realServer
}
