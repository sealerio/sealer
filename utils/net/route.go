// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	k8snet "k8s.io/apimachinery/pkg/util/net"
	k8sutilsnet "k8s.io/utils/net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/utils/exec"
)

const (
	RouteArg                    = "%s via %s dev %s metric 50"
	BackupAndDelStaticRouteFile = `if [ -f /etc/sysconfig/network-scripts/route-%s ]; then
  yes | cp /etc/sysconfig/network-scripts/route-%s /etc/sysconfig/network-scripts/.route-%s
  sed -i "/%s/d" /etc/sysconfig/network-scripts/route-%s
fi`
	AddStaticRouteFile = `cat /etc/sysconfig/network-scripts/route-%s|grep "%s" || echo "%s" >> /etc/sysconfig/network-scripts/route-%s`
)

const (
	RouteOK     = "ok"
	RouteFailed = "failed"
)

var ErrNotIPV4 = errors.New("IP addresses are not IPV4 rules")

type Route struct {
	Host    net.IP
	Gateway net.IP
}

func NewRouter(host, gateway net.IP) *Route {
	return &Route{
		Host:    host,
		Gateway: gateway,
	}
}

func CheckIsDefaultRoute(host net.IP) error {
	ok, err := isDefaultRouteIP(host)
	if err == nil && ok {
		_, err = common.StdOut.WriteString(RouteOK)
	}
	if err == nil && !ok {
		_, err = common.StdErr.WriteString(RouteFailed)
	}
	return err
}

// SetRoute ip route add $route
func (r *Route) SetRoute() error {
	if !k8sutilsnet.IsIPv4(r.Gateway) || !k8sutilsnet.IsIPv4(r.Host) {
		logrus.Warningf("%v, skip", ErrNotIPV4)
		return nil
	}
	err := addRouteGatewayViaHost(r.Host, r.Gateway, 50)
	if err != nil && !errors.Is(err, os.ErrExist) /* return if route already exist */ {
		return fmt.Errorf("failed to add %s route gateway via host err: %v", r.Host, err)
	}

	netInterface, err := GetHostNetInterface(r.Gateway)
	if err != nil {
		return err
	}
	if netInterface != "" {
		route := fmt.Sprintf(RouteArg, r.Host, r.Gateway, netInterface)
		_, err = exec.RunSimpleCmd(fmt.Sprintf(AddStaticRouteFile, netInterface, route, route, netInterface))
		if err != nil {
			return err
		}
	}
	logrus.Info(fmt.Sprintf("success to set route.(host:%s, gateway:%s)", r.Host, r.Gateway))
	return nil
}

// DelRoute ip route del $route
func (r *Route) DelRoute() error {
	if !k8sutilsnet.IsIPv4(r.Gateway) || !k8sutilsnet.IsIPv4(r.Host) {
		logrus.Warningf("%v, skip", ErrNotIPV4)
		return nil
	}
	err := delRouteGatewayViaHost(r.Host, r.Gateway)
	if err != nil && !errors.Is(err, syscall.ESRCH) /* return if route does not exist */ {
		return fmt.Errorf("failed to delete %s route gateway via host: %v", r.Host, err)
	}
	netInterface, err := GetHostNetInterface(r.Gateway)
	if err != nil {
		return err
	}
	if netInterface != "" {
		route := fmt.Sprintf(RouteArg, r.Host, r.Gateway, netInterface)
		out, err := exec.RunSimpleCmd(fmt.Sprintf(BackupAndDelStaticRouteFile, netInterface, netInterface, netInterface, route, netInterface))
		if err != nil {
			logrus.Info(out)
			return err
		}
	}
	logrus.Info(fmt.Sprintf("success to del route(host:%s, gateway:%s)", r.Host, r.Gateway))
	return nil
}

// isDefaultRouteIP return true if host equal default route ip host.
func isDefaultRouteIP(host net.IP) (bool, error) {
	netIP, err := k8snet.ChooseHostInterface()
	if err != nil {
		return false, fmt.Errorf("failed to get default route ip, err: %v", err)
	}
	return netIP.Equal(host), nil
}

func addRouteGatewayViaHost(host, gateway net.IP, priority int) error {
	Dst := &net.IPNet{
		IP:   host,
		Mask: net.CIDRMask(32, 32),
	}
	r := &netlink.Route{
		Dst:      Dst,
		Gw:       gateway,
		Priority: priority,
	}
	return netlink.RouteAdd(r)
}

func delRouteGatewayViaHost(host, gateway net.IP) error {
	Dst := &net.IPNet{
		IP:   host,
		Mask: net.CIDRMask(32, 32),
	}
	r := &netlink.Route{
		Dst: Dst,
		Gw:  gateway,
	}
	return netlink.RouteDel(r)
}

func IsIpv4(ip string) bool {
	arr := strings.Split(ip, ".")
	if len(arr) != 4 {
		return false
	}
	for _, v := range arr {
		if v == "" {
			return false
		}
		if len(v) > 1 && v[0] == '0' {
			return false
		}
		num := 0
		for _, c := range v {
			if c >= '0' && c <= '9' {
				num = num*10 + int(c-'0')
			} else {
				return false
			}
		}
		if num > 255 {
			return false
		}
	}
	return true
}
