package service

import (
	"encoding/base64"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/sagernet/sing-box/common/tls"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type ServerService struct {
	Runtime *Runtime
}

var systemInfoInterfaces = net.Interfaces

func (s *ServerService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (s *ServerService) GetStatus(request string) *map[string]interface{} {
	status := make(map[string]interface{}, 0)
	requests := strings.Split(request, ",")
	for _, req := range requests {
		switch req {
		case "cpu":
			status["cpu"] = s.GetCpuPercent()
		case "mem":
			status["mem"] = s.GetMemInfo()
		case "dsk":
			status["dsk"] = s.GetDiskInfo()
		case "dio":
			status["dio"] = s.GetDiskIO()
		case "swp":
			status["swp"] = s.GetSwapInfo()
		case "net":
			status["net"] = s.GetNetInfo()
		case "sys":
			status["sys"] = s.GetSystemInfo()
		case "sbd":
			status["sbd"] = s.GetSingboxInfo()
		case "db":
			status["db"] = s.GetDatabaseInfo()
		}
	}
	return &status
}

func (s *ServerService) GetCpuPercent() float64 {
	percents, err := cpu.Percent(0, false)
	if err != nil {
		logger.Warning("get cpu percent failed:", err)
		return 0
	} else {
		return percents[0]
	}
}

func (s *ServerService) GetMemInfo() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		logger.Warning("get virtual memory failed:", err)
	} else {
		info["current"] = memInfo.Used
		info["total"] = memInfo.Total
	}
	return info
}

func (s *ServerService) GetDiskInfo() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	diskInfo, err := disk.Usage("/")
	if err != nil {
		logger.Warning("get disk usage failed:", err)
	} else {
		info["current"] = diskInfo.Used
		info["total"] = diskInfo.Total
	}
	return info
}

func (s *ServerService) GetDiskIO() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	ioStats, err := disk.IOCounters()
	if err != nil {
		logger.Warning("get disk io counters failed:", err)
	} else if len(ioStats) > 0 {
		infoR, infoW := uint64(0), uint64(0)
		for _, ioStat := range ioStats {
			infoR += ioStat.ReadBytes
			infoW += ioStat.WriteBytes
		}
		info["read"] = infoR
		info["write"] = infoW
	} else {
		logger.Warning("can not find disk io counters")
	}
	return info
}

func (s *ServerService) GetSwapInfo() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	swapInfo, err := mem.SwapMemory()
	if err != nil {
		logger.Warning("get swap memory failed:", err)
	} else {
		info["current"] = swapInfo.Used
		info["total"] = swapInfo.Total
	}
	return info
}

func (s *ServerService) GetNetInfo() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	ioStats, err := net.IOCounters(false)
	if err != nil {
		logger.Warning("get io counters failed:", err)
	} else if len(ioStats) > 0 {
		ioStat := ioStats[0]
		info["sent"] = ioStat.BytesSent
		info["recv"] = ioStat.BytesRecv
		info["psent"] = ioStat.PacketsSent
		info["precv"] = ioStat.PacketsRecv
	} else {
		logger.Warning("can not find io counters")
	}
	return info
}

func (s *ServerService) GetSingboxInfo() map[string]interface{} {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	coreInstance := s.runtime().Core()
	isRunning := coreInstance != nil && coreInstance.IsRunning()
	uptime := uint32(0)
	if isRunning {
		if instance := coreInstance.GetInstance(); instance != nil {
			uptime = instance.Uptime()
		}
	}
	return map[string]interface{}{
		"running": isRunning,
		"stats": map[string]interface{}{
			"NumGoroutine": runtime.NumGoroutine(),
			"Alloc":        rtm.Alloc,
			"Uptime":       uptime,
		},
	}
}

func (s *ServerService) GetSystemInfo() map[string]interface{} {
	info := make(map[string]interface{}, 0)
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	info["appMem"] = rtm.Sys
	info["appThreads"] = runtime.NumGoroutine()
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info["cpuType"] = cpuInfo[0].ModelName
	}
	info["cpuCount"] = runtime.NumCPU()
	info["hostName"], _ = os.Hostname()
	info["appVersion"] = config.GetVersion()
	ipv4 := make([]string, 0)
	ipv6 := make([]string, 0)
	// get ip address
	netInterfaces, err := systemInfoInterfaces()
	if err != nil {
		logger.Warning("get net interfaces failed:", err)
	} else {
		for i := 0; i < len(netInterfaces); i++ {
			if !systemInterfaceUsable(netInterfaces[i].Flags) {
				continue
			}
			addrs := netInterfaces[i].Addrs

			for _, address := range addrs {
				publicAddress, ok := systemInfoPublicAddress(address.Addr)
				if !ok {
					continue
				}
				if strings.Contains(publicAddress, ".") {
					ipv4 = append(ipv4, publicAddress)
				} else {
					ipv6 = append(ipv6, publicAddress)
				}
			}
		}
	}
	info["ipv4"] = ipv4
	info["ipv6"] = ipv6
	info["bootTime"], _ = host.BootTime()

	return info
}

func systemInfoPublicAddress(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	var addr netip.Addr
	if prefix, err := netip.ParsePrefix(value); err == nil {
		addr = prefix.Addr()
	} else if parsed, err := netip.ParseAddr(value); err == nil {
		addr = parsed
	} else {
		return "", false
	}
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() ||
		addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsMulticast() ||
		addr.IsUnspecified() {
		return "", false
	}
	return value, true
}

func systemInterfaceUsable(flags []string) bool {
	up := false
	for _, flag := range flags {
		switch strings.ToLower(flag) {
		case "up":
			up = true
		case "loopback":
			return false
		}
	}
	return up
}

const (
	defaultLogCount = 10
	maxLogCount     = 500
	maxLogFilter    = 64
)

type LogQuery struct {
	Count  int
	Level  string
	Source string
	Filter string
}

func (s *ServerService) GetLogs(count string, level string) []string {
	logs, err := s.GetLogsFiltered(count, level, "", "")
	if err != nil {
		return nil
	}
	return logs
}

func (s *ServerService) GetLogsFiltered(count string, level string, source string, filter string) ([]string, error) {
	query, err := ParseLogQuery(count, level, source, filter)
	if err != nil {
		return nil, err
	}
	return logger.GetLogsFiltered(query.Count, query.Level, query.Source, query.Filter), nil
}

func ParseLogQuery(count string, level string, source string, filter string) (LogQuery, error) {
	parsedCount := defaultLogCount
	if count != "" {
		c, err := strconv.Atoi(count)
		if err != nil || c <= 0 {
			return LogQuery{}, common.NewError("invalid log count")
		}
		if c > maxLogCount {
			c = maxLogCount
		}
		parsedCount = c
	}
	if level == "" {
		level = "debug"
	}
	level = strings.ToLower(level)
	if !isValidLogLevel(level) {
		return LogQuery{}, common.NewError("invalid log level")
	}
	if source != "" && source != "panel" && source != "core" {
		return LogQuery{}, common.NewError("invalid log source")
	}
	if len(filter) > maxLogFilter || containsControlRune(filter) {
		return LogQuery{}, common.NewError("invalid log filter")
	}
	return LogQuery{
		Count:  parsedCount,
		Level:  level,
		Source: source,
		Filter: filter,
	}, nil
}

func isValidLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "notice", "warning", "error", "critical":
		return true
	default:
		return false
	}
}

func containsControlRune(value string) bool {
	for _, r := range value {
		if r == 0 || r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func (s *ServerService) GenKeypair(keyType string, options string) []string {
	if len(keyType) == 0 {
		return []string{"No keypair to generate"}
	}

	switch keyType {
	case "ech":
		return s.generateECHKeyPair(options)
	case "tls":
		return s.generateTLSKeyPair(options)
	case "reality":
		return s.generateRealityKeyPair()
	case "wireguard":
		return s.generateWireGuardKey(options)
	}

	return []string{"Failed to generate keypair"}
}

func (s *ServerService) generateECHKeyPair(serverName string) []string {
	configPem, keyPem, err := tls.ECHKeygenDefault(serverName)
	if err != nil {
		return []string{"Failed to generate ECH keypair: ", err.Error()}
	}
	return append(strings.Split(configPem, "\n"), strings.Split(keyPem, "\n")...)
}

func (s *ServerService) generateTLSKeyPair(serverName string) []string {
	privateKeyPem, publicKeyPem, err := tls.GenerateCertificate(nil, nil, time.Now, serverName, time.Now().AddDate(0, 12, 0))
	if err != nil {
		return []string{"Failed to generate TLS keypair: ", err.Error()}
	}
	return append(strings.Split(string(privateKeyPem), "\n"), strings.Split(string(publicKeyPem), "\n")...)
}

func (s *ServerService) generateRealityKeyPair() []string {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return []string{"Failed to generate Reality keypair: ", err.Error()}
	}
	publicKey := privateKey.PublicKey()
	return []string{"PrivateKey: " + base64.RawURLEncoding.EncodeToString(privateKey[:]), "PublicKey: " + base64.RawURLEncoding.EncodeToString(publicKey[:])}
}

func (s *ServerService) generateWireGuardKey(pk string) []string {
	if len(pk) > 0 {
		key, _ := wgtypes.ParseKey(pk)
		return []string{key.PublicKey().String()}
	}
	wgKeys, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return []string{"Failed to generate wireguard keypair: ", err.Error()}
	}
	return []string{"PrivateKey: " + wgKeys.String(), "PublicKey: " + wgKeys.PublicKey().String()}
}

func (s *ServerService) GetDatabaseInfo() map[string]int64 {
	info := make(map[string]int64, 0)
	db := database.GetDB()
	if db == nil {
		return nil
	}

	var clientsCount, inboundsCount, outboundsCount, servicesCount, endpointsCount, clientUp, clientDown int64

	db.Model(&model.Client{}).Count(&clientsCount)
	db.Model(&model.Inbound{}).Count(&inboundsCount)
	db.Model(&model.Outbound{}).Count(&outboundsCount)
	db.Model(&model.Service{}).Count(&servicesCount)
	db.Model(&model.Endpoint{}).Count(&endpointsCount)
	db.Model(&model.Client{}).Select("COALESCE(SUM(up+total_up),0)").Scan(&clientUp)
	db.Model(&model.Client{}).Select("COALESCE(SUM(down+total_down),0)").Scan(&clientDown)

	info["clients"] = clientsCount
	info["inbounds"] = inboundsCount
	info["outbounds"] = outboundsCount
	info["services"] = servicesCount
	info["endpoints"] = endpointsCount
	info["clientUp"] = clientUp
	info["clientDown"] = clientDown

	return info
}
