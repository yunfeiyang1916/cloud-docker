package subsystems

// MemorySubSystem 内存子系统
type MemorySubSystem struct {
}

// Name 名称
func (s *MemorySubSystem) Name() string {
	return "memory"
}
