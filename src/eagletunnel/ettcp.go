package eagletunnel

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"../eaglelib/src"
)

// ETTCP ET-TCP子协议的实现
type ETTCP struct {
}

// Send 发送请求
func (et *ETTCP) Send(e *NetArg) bool {
	switch ProxyStatus {
	case ProxySMART:
		el := ETLocation{}
		ok := el.Send(e)
		if !ok {
			// 位置解析失败，默认使用代理
			return et.sendTCPReq2Remote(e) == nil
		}
		if e.boolObj {
			// 启用直连
			err := et.sendTCPReq2Server(e)
			if err != nil {
				// 直连失败，换用代理
				return et.sendTCPReq2Remote(e) == nil
			}
			return true
		}
		// 不启用直连
		return et.sendTCPReq2Remote(e) == nil
	case ProxyENABLE:
		return et.sendTCPReq2Remote(e) == nil
	default:
		return false
	}
}

func (et *ETTCP) sendTCPReq2Remote(e *NetArg) error {
	err := connect2Relayer(e.tunnel)
	if err != nil {
		return err
	}
	req := FormatEtType(EtTCP) + " " + e.IP + " " + strconv.Itoa(e.Port)
	count, err := e.tunnel.WriteRight([]byte(req))
	if err != nil {
		return err
	}
	buffer := make([]byte, 1024)
	count, err = e.tunnel.ReadRight(buffer)
	if err != nil {
		return err
	}
	reply := string(buffer[:count])
	if reply != "ok" {
		err = errors.New("failed 2 connect 2 server by relayer")
	}
	return err
}

func (et *ETTCP) sendTCPReq2Server(e *NetArg) error {
	var ipe string
	ip := net.ParseIP(e.IP)
	if ip.To4() != nil {
		ipe = e.IP + ":" + strconv.Itoa(e.Port) // ipv4:port
	} else {
		ipe = "[" + e.IP + "]:" + strconv.Itoa(e.Port) // [ipv6]:port
	}
	conn, err := net.DialTimeout("tcp", ipe, 5*time.Second)
	if err != nil {
		return err
	}
	e.tunnel.Right = &conn
	e.tunnel.EncryptRight = false
	return err
}

// Handle 处理ET-TCP请求
func (et *ETTCP) Handle(req Request, tunnel *eaglelib.Tunnel) bool {
	reqs := strings.Split(req.RequestMsgStr, " ")
	if len(reqs) < 3 {
		return false
	}
	ip := reqs[1]
	_port := reqs[2]
	port, err := strconv.ParseInt(_port, 10, 32)
	if err != nil {
		return false
	}
	e := NetArg{IP: ip, Port: int(port), tunnel: tunnel}
	err = et.sendTCPReq2Server(&e)
	if err != nil {
		tunnel.WriteLeft([]byte("nok"))
		return false

	}
	_, err = tunnel.WriteLeft([]byte("ok"))
	return err == nil
}
