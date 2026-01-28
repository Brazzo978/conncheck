package tests

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type MTU struct {
	outDir string
	cfg    config.Config
}

func NewMTU(outDir string, cfg config.Config) *MTU {
	return &MTU{outDir: outDir, cfg: cfg}
}

func (m *MTU) Name() string {
	return "mtu_pmtu"
}

func (m *MTU) Run(ctx context.Context) model.TestResult {
	result := baseResult(m.Name())
	result.StartedAt = time.Now()
	result.Metrics["targets"] = joinList(m.cfg.Targets.MTUTargets)

	baseline, baselineEvidence := collectBaseline(m.outDir, m.cfg.Targets.MTUTargets)
	result.Evidence = append(result.Evidence, baselineEvidence...)
	if baseline.Interface != "" {
		result.Metrics["local_interface"] = baseline.Interface
	}
	if baseline.MTU > 0 {
		result.Metrics["local_mtu"] = fmt.Sprintf("%d", baseline.MTU)
	}
	if baseline.ConnType != "" {
		result.Metrics["connection_type"] = baseline.ConnType
	}
	if baseline.IPv4 != "" {
		result.Metrics["local_ipv4"] = baseline.IPv4
	}
	if baseline.IPv6 != "" {
		result.Metrics["local_ipv6"] = baseline.IPv6
	}
	if baseline.DNS != "" {
		result.Metrics["dns_servers"] = baseline.DNS
	}
	if baseline.Gateway != "" {
		result.Metrics["gateway"] = baseline.Gateway
	}

	targets := append([]string{}, m.cfg.Targets.MTUTargets...)
	if baseline.Gateway != "" && !containsTarget(targets, baseline.Gateway) {
		targets = append([]string{baseline.Gateway}, targets...)
	}
	if len(targets) == 0 {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "No MTU targets",
			Detail:   "No targets configured for PMTU checks.",
		})
		result.EndedAt = time.Now()
		return result
	}

	result.Status = StatusOK
	var pmtuValues []int
	var blackholeTargets []string
	var pmtuDetails []string
	var targetsTested []string

	for _, target := range targets {
		for _, stack := range []string{"ipv4", "ipv6"} {
			if !targetSupportsStack(target, stack) {
				continue
			}
			pmtuResult := runPMTUTest(ctx, m.outDir, target, stack)
			if pmtuResult.LogPath != "" {
				result.Evidence = append(result.Evidence, model.Evidence{
					Label: fmt.Sprintf("pmtu_%s_%s", sanitizeKey(target), stack),
					Path:  pmtuResult.LogPath,
				})
			}
			metricPrefix := fmt.Sprintf("pmtu_%s_%s", sanitizeKey(target), stack)
			if pmtuResult.PMTU > 0 {
				result.Metrics[metricPrefix] = fmt.Sprintf("%d", pmtuResult.PMTU)
				pmtuValues = append(pmtuValues, pmtuResult.PMTU)
				pmtuDetails = append(pmtuDetails, fmt.Sprintf("%s/%s=%d", target, stack, pmtuResult.PMTU))
			}
			result.Metrics[metricPrefix+"_frag_needed"] = boolLabel(pmtuResult.FragNeededSeen)
			if pmtuResult.BlackholeDetected {
				result.Metrics[metricPrefix+"_blackhole"] = "probable"
				blackholeTargets = append(blackholeTargets, fmt.Sprintf("%s/%s", target, stack))
			}
			if pmtuResult.Err != nil {
				result.Status = StatusWarn
				result.Findings = append(result.Findings, model.Finding{
					Severity: "WARN",
					Title:    "PMTU check failed",
					Detail:   fmt.Sprintf("PMTU check for %s (%s) failed: %s", target, stack, pmtuResult.Err.Error()),
				})
			}
			if pmtuResult.PMTU > 0 || pmtuResult.Err == nil {
				targetsTested = append(targetsTested, fmt.Sprintf("%s/%s", target, stack))
			}
		}
	}

	if len(pmtuValues) == 0 {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "PMTU not detected",
			Detail:   "No PMTU values could be measured from the configured targets.",
		})
		result.EndedAt = time.Now()
		return result
	}

	sort.Ints(pmtuValues)
	minPMTU := pmtuValues[0]
	result.Metrics["pmtu_min"] = fmt.Sprintf("%d", minPMTU)
	result.Metrics["pmtu_targets_tested"] = strings.Join(targetsTested, ",")
	if len(pmtuDetails) > 0 {
		result.Metrics["pmtu_details"] = strings.Join(pmtuDetails, "; ")
	}
	result.Metrics["pmtu_suggested_mtu"] = fmt.Sprintf("%d", minPMTU)

	if baseline.MTU > 0 && minPMTU > 0 && minPMTU < baseline.MTU {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "PMTU lower than interface MTU",
			Detail:   fmt.Sprintf("Detected PMTU %d while local MTU is %d. Possible clamping/PPPoE or tunnel overhead.", minPMTU, baseline.MTU),
		})
	}

	if len(blackholeTargets) > 0 {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Possible blackhole MTU",
			Detail:   fmt.Sprintf("Targets with DF loss and no ICMP fragmentation replies: %s.", strings.Join(blackholeTargets, ", ")),
		})
		result.Metrics["blackhole_mtu"] = "probable"
	} else {
		result.Metrics["blackhole_mtu"] = "no"
	}

	mssResult := collectMSS(ctx, targets)
	if mssResult.Err != nil {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "MSS observation failed",
			Detail:   mssResult.Err.Error(),
		})
	} else if mssResult.MSS > 0 {
		result.Metrics["mss_observed"] = fmt.Sprintf("%d", mssResult.MSS)
		result.Metrics["mss_class"] = mssResult.Class
		if mssResult.Class != "assente" {
			result.Status = StatusWarn
			result.Findings = append(result.Findings, model.Finding{
				Severity: "WARN",
				Title:    "MSS clamping detected",
				Detail:   fmt.Sprintf("Observed MSS %d (%s).", mssResult.MSS, mssResult.Class),
			})
		}
	}

	result.Metrics["mtu_health"] = scoreMTUHealth(minPMTU, len(blackholeTargets) > 0, mssResult.MSS, baseline.MTU)
	if result.Metrics["mtu_health"] != "OK" {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "MTU health warning",
			Detail:   fmt.Sprintf("MTU health rated %s. PMTU min %d, blackhole=%s, MSS=%s.", result.Metrics["mtu_health"], minPMTU, result.Metrics["blackhole_mtu"], result.Metrics["mss_class"]),
		})
	}

	result.Findings = append(result.Findings, suggestRemediations(minPMTU, len(blackholeTargets) > 0, mssResult.Class)...)
	result.EndedAt = time.Now()
	return result
}

type baselineInfo struct {
	Interface string
	MTU       int
	ConnType  string
	IPv4      string
	IPv6      string
	DNS       string
	Gateway   string
}

type pmtuResult struct {
	PMTU              int
	FragNeededSeen    bool
	BlackholeDetected bool
	LogPath           string
	Err               error
}

type mssInfo struct {
	MSS   int
	Class string
	Err   error
}

func collectBaseline(outDir string, targets []string) (baselineInfo, []model.Evidence) {
	if runtime.GOOS == "windows" {
		return collectBaselineWindows(outDir)
	}
	return collectBaselineUnix(outDir, targets)
}

func collectBaselineUnix(outDir string, targets []string) (baselineInfo, []model.Evidence) {
	baseline := baselineInfo{}
	var evidence []model.Evidence

	gateway, routeLog, err := DetectDefaultGateway(outDir)
	if routeLog != "" {
		evidence = append(evidence, model.Evidence{Label: "route", Path: routeLog})
	}
	if err == nil {
		baseline.Gateway = gateway
	}

	probeTarget := gateway
	if probeTarget == "" && len(targets) > 0 {
		probeTarget = targets[0]
	}
	if probeTarget == "" {
		probeTarget = "1.1.1.1"
	}

	output, logPath, err := sys.RunCommand(outDir, "ip", "route", "get", probeTarget)
	if logPath != "" {
		evidence = append(evidence, model.Evidence{Label: "ip_route_get", Path: logPath})
	}
	if err == nil {
		baseline.Interface = parseRouteInterface(output)
		baseline.IPv4 = parseRouteSource(output)
	}

	if baseline.Interface != "" {
		linkOut, linkLog, err := sys.RunCommand(outDir, "ip", "link", "show", "dev", baseline.Interface)
		if linkLog != "" {
			evidence = append(evidence, model.Evidence{Label: "ip_link", Path: linkLog})
		}
		if err == nil {
			baseline.MTU = parseInterfaceMTU(linkOut)
		}

		ip6Out, ip6Log, err := sys.RunCommand(outDir, "ip", "-6", "addr", "show", "dev", baseline.Interface)
		if ip6Log != "" {
			evidence = append(evidence, model.Evidence{Label: "ip6_addr", Path: ip6Log})
		}
		if err == nil {
			baseline.IPv6 = parseIPv6Addr(ip6Out)
		}
		baseline.ConnType = detectInterfaceType(baseline.Interface)
	}

	baseline.DNS = readDNSResolvers()
	return baseline, evidence
}

func collectBaselineWindows(outDir string) (baselineInfo, []model.Evidence) {
	baseline := baselineInfo{}
	var evidence []model.Evidence

	gateway, routeLog, err := DetectDefaultGateway(outDir)
	if routeLog != "" {
		evidence = append(evidence, model.Evidence{Label: "route", Path: routeLog})
	}
	if err == nil {
		baseline.Gateway = gateway
	}

	ipconfigOut, ipconfigLog, err := sys.RunCommand(outDir, "ipconfig", "/all")
	if ipconfigLog != "" {
		evidence = append(evidence, model.Evidence{Label: "ipconfig", Path: ipconfigLog})
	}
	if err == nil {
		section := pickWindowsAdapterSection(ipconfigOut, baseline.Gateway)
		baseline.Interface = section.Name
		baseline.IPv4 = section.IPv4
		baseline.IPv6 = section.IPv6
		baseline.DNS = section.DNS
		if baseline.ConnType == "" {
			baseline.ConnType = classifyWindowsInterface(section.Name)
		}
	}

	if baseline.Interface != "" {
		subOut, subLog, err := sys.RunCommand(outDir, "netsh", "interface", "ipv4", "show", "subinterfaces")
		if subLog != "" {
			evidence = append(evidence, model.Evidence{Label: "netsh_subinterfaces", Path: subLog})
		}
		if err == nil {
			baseline.MTU = parseWindowsSubinterfaceMTU(subOut, baseline.Interface)
		}
	}

	if baseline.ConnType == "" && baseline.Interface != "" {
		baseline.ConnType = classifyWindowsInterface(baseline.Interface)
	}
	return baseline, evidence
}

func parseRouteInterface(output string) string {
	fields := strings.Fields(output)
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "dev" {
			return fields[i+1]
		}
	}
	return ""
}

func parseRouteSource(output string) string {
	fields := strings.Fields(output)
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "src" {
			return fields[i+1]
		}
	}
	return ""
}

func parseInterfaceMTU(output string) int {
	re := regexp.MustCompile(`mtu\s+(\d+)`)
	if m := re.FindStringSubmatch(output); len(m) == 2 {
		mtu, _ := strconv.Atoi(m[1])
		return mtu
	}
	return 0
}

func parseIPv6Addr(output string) string {
	re := regexp.MustCompile(`inet6\s+([0-9a-fA-F:]+)`)
	if m := re.FindStringSubmatch(output); len(m) == 2 {
		return m[1]
	}
	return ""
}

func detectInterfaceType(iface string) string {
	if iface == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join("/sys/class/net", iface, "wireless")); err == nil {
		return "Wi-Fi"
	}
	if strings.HasPrefix(iface, "wl") || strings.Contains(strings.ToLower(iface), "wifi") {
		return "Wi-Fi"
	}
	if strings.HasPrefix(iface, "en") || strings.HasPrefix(iface, "eth") {
		return "Ethernet"
	}
	return "Unknown"
}

func readDNSResolvers() string {
	content, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return ""
	}
	var servers []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}
	return strings.Join(servers, ",")
}

type windowsAdapterSection struct {
	Name string
	IPv4 string
	IPv6 string
	DNS  string
}

func pickWindowsAdapterSection(output, gateway string) windowsAdapterSection {
	sections := splitWindowsIPConfig(output)
	for _, section := range sections {
		if gateway != "" && strings.Contains(section, gateway) {
			return parseWindowsSection(section)
		}
	}
	if len(sections) > 0 {
		return parseWindowsSection(sections[0])
	}
	return windowsAdapterSection{}
}

func splitWindowsIPConfig(output string) []string {
	blocks := strings.Split(output, "\n\n")
	var sections []string
	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "adapter") {
			sections = append(sections, trimmed)
		}
	}
	return sections
}

func parseWindowsSection(section string) windowsAdapterSection {
	lines := strings.Split(section, "\n")
	result := windowsAdapterSection{}
	if len(lines) > 0 {
		result.Name = strings.TrimSpace(strings.TrimSuffix(lines[0], ":"))
	}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(line, "IPv4 Address") || strings.HasPrefix(line, "IPv4 Address."):
			result.IPv4 = parseWindowsValue(line)
		case strings.HasPrefix(line, "IPv6 Address") || strings.HasPrefix(line, "IPv6 Address."):
			result.IPv6 = parseWindowsValue(line)
		case strings.HasPrefix(line, "DNS Servers") || strings.HasPrefix(line, "DNS Servers."):
			result.DNS = parseWindowsValue(line)
			for j := i + 1; j < len(lines); j++ {
				candidate := strings.TrimSpace(lines[j])
				if candidate == "" || strings.Contains(candidate, ":") {
					break
				}
				result.DNS = strings.Join([]string{result.DNS, strings.TrimSpace(candidate)}, ",")
			}
		}
	}
	result.Name = strings.TrimSpace(strings.TrimPrefix(result.Name, "Ethernet adapter"))
	result.Name = strings.TrimSpace(strings.TrimPrefix(result.Name, "Wireless LAN adapter"))
	return result
}

func parseWindowsValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	value := strings.TrimSpace(parts[1])
	return strings.TrimSuffix(value, "(Preferred)")
}

func classifyWindowsInterface(name string) string {
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "wi-fi") || strings.Contains(nameLower, "wireless") {
		return "Wi-Fi"
	}
	if strings.Contains(nameLower, "ethernet") {
		return "Ethernet"
	}
	return "Unknown"
}

func parseWindowsSubinterfaceMTU(output, iface string) int {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		name := strings.Join(fields[4:], " ")
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(iface)) {
			mtu, _ := strconv.Atoi(fields[0])
			return mtu
		}
	}
	return 0
}

func runPMTUTest(ctx context.Context, outDir, target, stack string) pmtuResult {
	maxPayload := 1472
	if stack == "ipv6" {
		maxPayload = 1452
	}
	low := 0
	high := maxPayload
	lastSuccess := -1
	fragSeen := false
	blackhole := false
	var lastLog string

	for low <= high {
		mid := (low + high) / 2
		res := pingDF(ctx, outDir, target, stack, mid)
		if res.LogPath != "" {
			lastLog = res.LogPath
		}
		if res.Success {
			lastSuccess = mid
			low = mid + 1
			continue
		}
		if res.FragNeeded {
			fragSeen = true
		}
		high = mid - 1
	}

	if lastSuccess < 0 {
		return pmtuResult{PMTU: 0, FragNeededSeen: fragSeen, BlackholeDetected: blackhole, LogPath: lastLog, Err: errors.New("no successful DF ping")}
	}
	if !fragSeen && lastSuccess < maxPayload {
		blackhole = true
	}
	return pmtuResult{PMTU: payloadToMTU(lastSuccess, stack), FragNeededSeen: fragSeen, BlackholeDetected: blackhole, LogPath: lastLog}
}

type pingResult struct {
	Success    bool
	FragNeeded bool
	LogPath    string
}

func pingDF(ctx context.Context, outDir, target, stack string, payload int) pingResult {
	var output, logPath string
	var err error
	if runtime.GOOS == "windows" {
		args := []string{"-n", "1", "-f", "-l", fmt.Sprintf("%d", payload)}
		if stack == "ipv4" {
			args = append([]string{"-4"}, args...)
		} else {
			args = append([]string{"-6"}, args...)
		}
		args = append(args, target)
		output, logPath, err = sys.RunCommand(outDir, "ping", args...)
	} else {
		args := []string{"-c", "1", "-M", "do", "-s", fmt.Sprintf("%d", payload)}
		if stack == "ipv4" {
			args = append([]string{"-4"}, args...)
		} else {
			args = append([]string{"-6"}, args...)
		}
		args = append(args, target)
		output, logPath, err = sys.RunCommand(outDir, "ping", args...)
	}
	result := pingResult{LogPath: logPath}
	if err == nil {
		result.Success = true
		return result
	}
	if containsFragNeeded(output) {
		result.FragNeeded = true
	}
	return result
}

func containsFragNeeded(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "frag") || strings.Contains(output, "fragmentation needed") || strings.Contains(output, "packet needs to be fragmented") || strings.Contains(output, "message too long")
}

func payloadToMTU(payload int, stack string) int {
	if stack == "ipv6" {
		return payload + 48
	}
	return payload + 28
}

func targetSupportsStack(target, stack string) bool {
	ip := net.ParseIP(strings.Trim(target, "[]"))
	if ip == nil {
		return true
	}
	if stack == "ipv4" {
		return ip.To4() != nil
	}
	return ip.To4() == nil
}

func sanitizeKey(target string) string {
	var b strings.Builder
	for _, r := range target {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	return b.String()
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func containsTarget(targets []string, candidate string) bool {
	for _, t := range targets {
		if t == candidate {
			return true
		}
	}
	return false
}

func collectMSS(ctx context.Context, targets []string) mssInfo {
	for _, target := range targets {
		if target == "" {
			continue
		}
		stack := "tcp4"
		if ip := net.ParseIP(strings.Trim(target, "[]")); ip != nil && ip.To4() == nil {
			stack = "tcp6"
		}
		mss, err := observeMSS(ctx, target, stack)
		if err != nil {
			continue
		}
		return mssInfo{MSS: mss, Class: classifyMSS(mss)}
	}
	return mssInfo{Err: errors.New("unable to observe MSS from targets")}
}

func observeMSS(ctx context.Context, target, network string) (int, error) {
	if runtime.GOOS == "windows" {
		return 0, errors.New("MSS observation not supported on Windows")
	}
	address, err := normalizeAddress(target, "443")
	if err != nil {
		return 0, err
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, errors.New("non-TCP connection")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var mss int
	var sysErr error
	controlErr := rawConn.Control(func(fd uintptr) {
		mss, sysErr = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG)
	})
	if controlErr != nil {
		return 0, controlErr
	}
	if sysErr != nil {
		return 0, sysErr
	}
	if mss <= 0 {
		return 0, errors.New("invalid MSS result")
	}
	return mss, nil
}

func normalizeAddress(target, defaultPort string) (string, error) {
	if target == "" {
		return "", errors.New("empty target")
	}
	if strings.Contains(target, ":") {
		if host, port, err := net.SplitHostPort(target); err == nil {
			if host == "" {
				return "", errors.New("empty host")
			}
			if port == "" {
				port = defaultPort
			}
			return net.JoinHostPort(host, port), nil
		}
		if strings.Count(target, ":") > 1 && !strings.HasPrefix(target, "[") {
			return net.JoinHostPort(target, defaultPort), nil
		}
	}
	return net.JoinHostPort(target, defaultPort), nil
}

func classifyMSS(mss int) string {
	switch {
	case mss >= 1456:
		return "assente"
	case mss >= 1440:
		return "pppoe_sospetto"
	case mss >= 1360:
		return "basso"
	default:
		return "aggressivo"
	}
}

func scoreMTUHealth(pmtuMin int, blackhole bool, mss int, ifaceMTU int) string {
	if pmtuMin == 0 {
		return "Warning"
	}
	if blackhole {
		return "Bad"
	}
	if pmtuMin < 1400 {
		return "Bad"
	}
	if ifaceMTU > 0 && pmtuMin < ifaceMTU {
		return "Warning"
	}
	if mss > 0 && mss < 1360 {
		return "Warning"
	}
	return "OK"
}

func suggestRemediations(pmtuMin int, blackhole bool, mssClass string) []model.Finding {
	var findings []model.Finding
	if pmtuMin > 0 && pmtuMin < 1500 {
		findings = append(findings, model.Finding{
			Severity: "INFO",
			Title:    "Suggested MTU",
			Detail:   fmt.Sprintf("Suggested effective MTU: %d (based on PMTU min).", pmtuMin),
		})
	}
	if blackhole {
		findings = append(findings, model.Finding{
			Severity: "WARN",
			Title:    "Possible ICMP blocking",
			Detail:   "Some paths likely block ICMP fragmentation-needed replies (blackhole MTU). This can cause pages not loading or unstable VPNs.",
		})
	}
	if mssClass == "pppoe_sospetto" {
		findings = append(findings, model.Finding{
			Severity: "WARN",
			Title:    "PPPoE overhead suspected",
			Detail:   "Observed MSS suggests PPPoE/overhead (MTU ~1492).",
		})
	}
	if mssClass == "basso" || mssClass == "aggressivo" {
		findings = append(findings, model.Finding{
			Severity: "WARN",
			Title:    "MSS clamping",
			Detail:   "Observed MSS is lower than expected; a router or upstream network is clamping MSS.",
		})
	}
	return findings
}
