package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// Create 创建网络
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	// 解析网关ip和网络ip段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	// 初始化网络对象
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	// 配置linux bridge
	err := d.initBridge(n)

	if err != nil {
		log.Errorf("error init bridge: %v", err)
	}

	return n, err
}

func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// 1.创建网桥虚拟设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge： %s, Error: %v", bridgeName, err)
	}

	// 2.设置网桥设备的地址和路由
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}
	// 3.启动网桥设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s, Error: %v", bridgeName, err)
	}

	// 4.设置iptabels的SNAT规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}

// 创建网桥设备
func createBridgeInterface(bridgeName string) error {
	// 是否已经存在了同名的网桥设备
	_, err := net.InterfaceByName(bridgeName)
	// err==nil表示已经存在同名设备，如果是非no such错误表示调用失败
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	// 初始化一个netlink的link基础对象，link的名字即网桥虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 创建网桥对象
	br := &netlink.Bridge{LinkAttrs: la}
	// 创建虚拟网络设备，相当于ip link add xxxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// 设置一个网络接口的ip地址，例如：setInterfaceIP("testbridge","192.168.0.1/24)
func setInterfaceIP(name, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		// 找到需要设置的网络接口
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
	// netlink.ParseIPNet 是对net.ParseCIDR的封装
	// ipNet包含网段的信息（192.168.0.0/24）,也包含了原始的ip 192.168.0.1
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{
		IPNet: ipNet,
		Label: "",
		Flags: 0,
		Scope: 0,
		Peer:  nil,
	}
	// 通过netlink.AddrAdd给网络接口配置地址，相当于ip addr add xxx的命令
	// 同时如果配置了地址所在的网段信息，例如192.168.0.0/24,还会配置路由表192.168.0.0/24转发到这个网桥的网络接口上
	return netlink.AddrAdd(iface, addr)
}

// 设置网络接口为UP状态
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}
	// 等价于ip link set xxx up命令
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// 设置iptables对应bridge的MASQUERADE（伪装）规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 由于go语言没有直接操控iptables的框，所以需要通过命令的方式来配置
	// 创建iptables的命令
	// iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}

// Delete 删除网络
func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// Connect 连接一个网络和网络端点
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 创建Veth接口的配置
	la := netlink.NewLinkAttrs()
	// 由于linux接口名的限制，名字取endpoint ID的前5位
	la.Name = endpoint.ID[:5]
	// 通过设置Veth接口的master属性，设置这个veth的一端挂载到对应的linux网桥上
	la.MasterIndex = br.Attrs().Index

	// 创建veth对象，通过PeerName配置veth另外一端的接口名
	// 配置veth另外一端的名字cif-{endpoint ID的前5位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}
	// 调用LinkAdd创建这个veth接口
	// 因为上面指定了link的MasterIndex是网络对应的Linux网桥
	// 所以veth的一端就已经挂载到了网络对应的Linux网桥上了
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}
	// 设置veth启动，相当于ip link set xxx up 命令
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}
