package proc

import (
	"encoding/json"
	"os"
)

const (
	SysProcPath  = "/proc/sysinfo_so1_202203009"
	ContProcPath = "/proc/continfo_so1_202203009"
)

type SysSnapshot struct {
	Totalram  uint64    `json:"Totalram"`
	Freeram   uint64    `json:"Freeram"`
	Procs     int       `json:"Procs"`
	Processes []SysProc `json:"Processes"`
}
type SysProc struct {
	PID     int     `json:"PID"`
	Name    string  `json:"Name"`
	State   string  `json:"State"`
	Cmdline string  `json:"Cmdline"`
	VSZ     uint64  `json:"vsz"`
	RSS     uint64  `json:"rss"`
	MemPct  float64 `json:"Memory_Usage"`
	CPUPct  float64 `json:"CPU_Usage"`
}

type ContSnapshot struct {
	Totalram  uint64     `json:"Totalram"`
	Freeram   uint64     `json:"Freeram"`
	Processes []ContProc `json:"Processes"`
}
type ContProc struct {
	ShimPID     int     `json:"ShimPID"`
	ShimName    string  `json:"ShimName"`
	ContainerID string  `json:"ContainerID"`
	PID         int     `json:"PID"`
	Name        string  `json:"Name"`
	Cmdline     string  `json:"Cmdline"`
	VSZ         uint64  `json:"vsz"`
	RSS         uint64  `json:"rss"`
	MemPct      float64 `json:"Memory_Usage"`
	CPUPct      float64 `json:"CPU_Usage"`
}

// ReadJSON lee y deserializa JSON desde path a out.
func ReadJSON[T any](path string, out *T) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
