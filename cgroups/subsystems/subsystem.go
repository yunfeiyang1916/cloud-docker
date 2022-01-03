package subsystems

// ResourceConfig 用于传递资源限制的结构体
type ResourceConfig struct {
	// 内存限制
	MemoryLimit string
	// cpu时间片权重
	CpuShare string
	// cpu核心数
	CpuSet string
}

// SubSystem 接口，这里将cgroup抽象成了path，原因是cgroup在hierarchy路径，便是虚拟文件中的虚拟路径
type SubSystem interface {
	// Name 子系统名称，比如cpu、memory
	Name() string
	// Set 设置cgroup在这个子系统中的资源限制
	Set(path string, res *ResourceConfig) error
	// Apply 将进程添加到某个cgroup中
	Apply(path string, pid int) error
	// Remove 移除某个cgroup
	Remove(path string) error
}

var ()
