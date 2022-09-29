//go:build linux
// +build linux

package network

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/yunfeiyang1916/cloud-docker/container"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/cloud-docker/network/network/"
	// 网络驱动map
	drivers  = map[string]NetworkDriver{}
	networks = map[string]*Network{}
)

// Endpoint 网络端点
type Endpoint struct {
	ID string `json:"id"`
	// 虚拟网络设备
	Device     netlink.Veth     `json:"dev"`
	IPAddress  net.IP           `json:"ip"`
	MacAddress net.HardwareAddr `json:"mac"`
	// 端口映射
	PortMapping []string `json:"portmapping"`
	// 网络
	Network *Network
}

// Network 网络
type Network struct {
	// 网络名
	Name string
	// 地址段
	IpRange *net.IPNet
	// 网络驱动名
	Driver string
}

// 保存网络信息
func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	// 文件名是网络的名字
	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error：", err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error：", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error：", err)
		return err
	}
	return nil
}

// 移除网络信息
func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}

// 读取网络配置信息
func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Error load nw info", err)
		return err
	}
	return nil
}

// NetworkDriver 网络驱动
type NetworkDriver interface {
	// Name 驱动名
	Name() string
	// Create 创建网络
	Create(subnet, name string) (*Network, error)
	// Delete 删除网络
	Delete(network Network) error
	// Connect 连接
	Connect(network *Network, endpoint *Endpoint) error
	// Disconnect 断开连接
	Disconnect(network Network, endpoint *Endpoint) error
}

func Init() error {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 判断网络的配置目录是否存在，不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}
	// 检查网络配置目录中的所有文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		// 跳过目录
		if info.IsDir() {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}
		// 将网络的配置信息加入到networks字典中
		networks[nwName] = nw
		return nil
	})

	//logrus.Infof("networks: %v", networks)

	return nil
}

// ListNetwork 展示网络列表
func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 将网段字符串转成net.IPNet对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关ip,获取到的网段中第一个ip作为网关ip
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp
	// 创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	// 保存网络信息，将网络的信息保存在文件 系统中，以便查询和在网络上连接网络端点
	return nw.dump(defaultNetworkPath)
}

func DeleteNetwork(networkName string) error {
	// 查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}
	// 通过IPAM的实例释放网关的IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error Remove Network gateway ip: %s", err)
	}
	// 删除网络设备与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %s", err)
	}
	// 从网络的配置目录删除该网络的对应的配置文件
	return nw.remove(defaultNetworkPath)
}

// 将容器的网络端点加入到容器的网络空间中
// 并锁定当前程序所执行的线程,使当前线程进入到容器的网络空间
// 返回值是一个函数指针，执行这个返回函数才会退出容器的网络空间，回归到宿主机的网络空间
func enterContainerNetns(enLink *netlink.Link, info *container.ContainerInfo) func() {
	// 找到容器的网络命名空间
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
	// 而ContainerInfo中的PID就是容器在宿主机上映射的进程ID
	// 它对应的/proc/[pid]/ns/net就是容器内部的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}
	// 取到文件的文件描述符
	nsFD := f.Fd()

	// 锁定当前程序所执行的线程
	// 如果不锁定操作系统线程的话协程可能会被调度到别的线程上去，就不能保证一直在所需要的网络空间中了
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

// 到容器的命名空间配置容器网络设备IP地址
func configEndpointIpAddressAndRoute(ep *Endpoint, info *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	// 将容器的网络端点加入到容器的网络空间中,并使这个函数下面的操作都在这个网络空间中执行
	// 执行完函数后，恢复为默认的网络空间
	defer enterContainerNetns(&peerLink, info)()

	// 获取到容器的IP地址及网段，用于配置容器内部接口地址
	// 比如容器IP是192.168.1.2，而网段是192.168.1.0/24
	// 那么这里产出的IP字符串就是192.168.1.2/24,用于容器内veth端点配置
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress
	// 设置容器内veth端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	// 启动容器内的veth端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	// 启动lo网卡，以保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	// 设置容器内的外部请求都通过容器内的veth端点访问
	// 0.0.0.0/0 表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	// 相当于route add -net 0.0.0.0/0 gw {网桥地址} dev {容器内的veth端点设备}
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}
	// 添加路由到容器的网络空间，相当于route add 命令
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

// 配置端口映射
func configPortMapping(ep *Endpoint, info *container.ContainerInfo) error {
	// 遍历容器端口映射列表
	for _, pm := range ep.PortMapping {
		// 分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}
		// 在iptables的PREROUTING中添加DNAT规则，将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

// Connect 连接容器到之前创建的网络
func Connect(networkName string, info *container.ContainerInfo) error {
	// 从networks字典中取到容器连接的网络的信息，networks字典中保存了当前已经创建的网络
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}
	// 分配容器IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}
	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: info.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}
	// 到容器的命名空间配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, info); err != nil {
		return err
	}
	// 配置容器到宿主机的端口映射
	return configPortMapping(ep, info)
}

func Disconnect(networkName string, info *container.ContainerInfo) error {
	return nil
}
