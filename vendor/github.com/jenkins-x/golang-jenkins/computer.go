package gojenkins

type ComputerObject struct {
	BusyExecutors  int        `json:"busyExecutors"`
	Computers      []Computer `json:"computer"`
	DisplayName    string     `json:"displayName"`
	TotalExecutors int        `json:"totalExecutors"`
}

type Computer struct {
	Actions             []struct{} `json:"actions"`
	DisplayName         string     `json:"displayName"`
	Executors           []struct{} `json:"executors"`
	Idle                bool       `json:"idle"`
	JnlpAgent           bool       `json:"jnlpAgent"`
	LaunchSupported     bool       `json:"launchSupported"`
	ManualLaunchAllowed bool       `json:"manualLaunchAllowed"`
	MonitorData         struct {
		SwapSpaceMonitor struct {
			AvailablePhysicalMemory int64 `json:"availablePhysicalMemory"`
			AvailableSwapSpace      int64 `json:"availableSwapSpace"`
			TotalPhysicalMemory     int64 `json:"totalPhysicalMemory"`
			TotalSwapSpace          int64 `json:"totalSwapSpace"`
		} `json:"hudson.node_monitors.SwapSpaceMonitor"`
		TemporarySpaceMonitor struct {
			Timestamp int64  `json:"timestamp"`
			Path      string `json:"path"`
			Size      int64  `json:"size"`
		} `json:"hudson.node_monitors.TemporarySpaceMonitor"`
		DiskSpaceMonitor struct {
			Timestamp int64  `json:"timestamp"`
			Path      string `json:"path"`
			Size      int64  `json:"size"`
		} `json:"hudson.node_monitors.DiskSpaceMonitor"`
		ArchitectureMonitor string `json:"hudson.node_monitors.ArchitectureMonitor"`
		ResponseTimeMonitor struct {
			Timestamp int64 `json:"timestamp"`
			Average   int64 `json:"average"`
		} `json:"hudson.node_monitors.ResponseTimeMonitor"`
		ClockMonitor struct {
			Diff int64 `json:"diff"`
		} `json:"hudson.node_monitors.ClockMonitor"`
	} `json:"monitorData"`
	NumExecutors       int    `json:"numExecutors"`
	Offline            bool   `json:"offline"`
	OfflineCauseReason string `json:"offlineCauseReason"`
	TemporarilyOffline bool   `json:"temporarilyOffline"`
}
