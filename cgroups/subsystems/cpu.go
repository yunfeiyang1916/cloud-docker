package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// CpuSubSystem cpu子系统
type CpuSubSystem struct {
}

// Name 名称
func (s *CpuSubSystem) Name() string {
	return "cpu"
}

// Set 设置cgroupPath对应的cgroup的cpu资源限制
func (s *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	// GetCgroupPath 的作用是获取当前subsystem在虚拟文件系统中的路径
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	if res.CpuShare != "" {
		// 设置这个cgroup的内存限制，即将限制写入到cgroup对应目录的cpu.shares文件中
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpu.shares"), []byte(res.MemoryLimit), 0644); err != nil {
			return fmt.Errorf("set cgroup memory fail %v", err)
		}
	}
	return nil
}

// Remove 删除cgroupPath对应的cgroup
func (s *CpuSubSystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	// 删除cgroup便是删除对应的cgroupPath的目录
	return os.RemoveAll(subsysCgroupPath)
}

// Apply 将一个进程加入到cgroupPath对应的cgroup中
func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
	// 把进程的pid写到cgroup的虚拟文件系统对应目录写的"task"文件中
	if err = ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}
