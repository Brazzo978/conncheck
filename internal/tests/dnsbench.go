package tests

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type DNSBench struct {
	outDir string
	cfg    config.Config
}

func NewDNSBench(outDir string, cfg config.Config) *DNSBench {
	return &DNSBench{outDir: outDir, cfg: cfg}
}

func (d *DNSBench) Name() string {
	return "dns_benchmark"
}

func (d *DNSBench) Run(ctx context.Context) model.TestResult {
	result := baseResult(d.Name())
	result.StartedAt = time.Now()
	result.Status = StatusSkipped

	domains := d.cfg.Targets.DNSDomains
	if len(domains) == 0 {
		domains = []string{"www.google.com", "www.cloudflare.com", "www.wikipedia.org"}
	}

	systemServers, evidence, err := systemDNSServers(d.outDir)
	if evidence != nil {
		result.Evidence = append(result.Evidence, *evidence)
	}
	if err != nil {
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Unable to read system DNS servers",
			Detail:   err.Error(),
		})
	}

	configServers := d.cfg.Targets.DNSServers
	allServers := uniqueServers(append(append([]string{}, systemServers...), configServers...))

	queriesPerDomain := d.cfg.Targets.DNSQueriesPerDomain
	if queriesPerDomain <= 0 {
		queriesPerDomain = 3
	}

	result.Metrics["dns_domains"] = joinList(domains)
	result.Metrics["dns_queries_per_domain"] = strconv.Itoa(queriesPerDomain)
	result.Metrics["dns_timeout_ms"] = "2000"
	result.Metrics["dhcp_dns_servers"] = joinList(systemServers)
	result.Metrics["config_dns_servers"] = joinList(configServers)
	result.Metrics["dns_servers"] = joinList(allServers)

	if len(allServers) == 0 {
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "No DNS servers available",
			Detail:   "Provide DNS servers in config or ensure DHCP provides resolvers.",
		})
		result.EndedAt = time.Now()
		return result
	}

	type serverResult struct {
		server  string
		success int
		fail    int
		avgMs   float64
		hasAvg  bool
	}

	results := make(chan serverResult, len(allServers))
	var wg sync.WaitGroup

	for _, server := range allServers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			serverSuccess := 0
			serverFail := 0
			var totalLatency time.Duration
			for _, domain := range domains {
				for i := 0; i < queriesPerDomain; i++ {
					latency, err := dnsLookupLatency(ctx, server, domain, 2*time.Second)
					if err != nil {
						serverFail++
						continue
					}
					serverSuccess++
					totalLatency += latency
				}
			}

			result := serverResult{
				server:  server,
				success: serverSuccess,
				fail:    serverFail,
			}
			if serverSuccess > 0 {
				result.avgMs = float64(totalLatency.Milliseconds()) / float64(serverSuccess)
				result.hasAvg = true
			}
			results <- result
		}(server)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	totalSuccess := 0
	totalFail := 0
	for serverResult := range results {
		totalSuccess += serverResult.success
		totalFail += serverResult.fail
		if serverResult.hasAvg {
			result.Metrics[fmt.Sprintf("dns_avg_ms.%s", serverResult.server)] = fmt.Sprintf("%.2f", serverResult.avgMs)
		}
		result.Metrics[fmt.Sprintf("dns_success.%s", serverResult.server)] = strconv.Itoa(serverResult.success)
		result.Metrics[fmt.Sprintf("dns_fail.%s", serverResult.server)] = strconv.Itoa(serverResult.fail)
	}

	result.Metrics["dns_success_total"] = strconv.Itoa(totalSuccess)
	result.Metrics["dns_fail_total"] = strconv.Itoa(totalFail)

	if totalSuccess == 0 {
		result.Status = StatusFail
		result.Findings = append(result.Findings, model.Finding{
			Severity: "FAIL",
			Title:    "DNS benchmark failed",
			Detail:   "All DNS queries failed for every server.",
		})
	} else if totalFail > 0 {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Partial DNS failures detected",
			Detail:   fmt.Sprintf("%d queries failed out of %d.", totalFail, totalSuccess+totalFail),
		})
	} else {
		result.Status = StatusOK
	}

	result.EndedAt = time.Now()
	return result
}

func dnsLookupLatency(ctx context.Context, server, domain string, timeout time.Duration) (time.Duration, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: timeout}
			return dialer.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
		},
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	_, err := resolver.LookupIPAddr(queryCtx, domain)
	return time.Since(start), err
}

func systemDNSServers(outDir string) ([]string, *model.Evidence, error) {
	if runtime.GOOS == "windows" {
		output, logPath, err := sys.RunCommand(outDir, "ipconfig", "/all")
		if err != nil {
			return nil, &model.Evidence{Label: "ipconfig_dns", Path: logPath, Note: err.Error()}, err
		}
		return parseIPConfigDNSServers(output), &model.Evidence{Label: "ipconfig_dns", Path: logPath}, nil
	}

	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil, nil, err
	}
	servers := parseResolvConfDNSServers(string(data))
	return servers, &model.Evidence{Label: "resolv_conf", Path: "/etc/resolv.conf"}, nil
}

func parseIPConfigDNSServers(output string) []string {
	lines := strings.Split(output, "\n")
	var servers []string
	inBlock := false
	for _, line := range lines {
		if strings.Contains(line, "DNS Servers") {
			inBlock = true
			servers = append(servers, extractIPs(line)...)
			continue
		}
		if inBlock {
			if strings.TrimSpace(line) == "" {
				inBlock = false
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				servers = append(servers, extractIPs(line)...)
				continue
			}
			inBlock = false
		}
	}
	return uniqueServers(servers)
}

func parseResolvConfDNSServers(contents string) []string {
	var servers []string
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}
		if net.ParseIP(fields[1]) != nil {
			servers = append(servers, fields[1])
		}
	}
	return uniqueServers(servers)
}

func extractIPs(line string) []string {
	fields := strings.Fields(line)
	servers := []string{}
	for _, field := range fields {
		cleaned := strings.Trim(field, ",;")
		if net.ParseIP(cleaned) != nil {
			servers = append(servers, cleaned)
		}
	}
	return servers
}

func uniqueServers(servers []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, server := range servers {
		if server == "" {
			continue
		}
		if seen[server] {
			continue
		}
		seen[server] = true
		unique = append(unique, server)
	}
	return unique
}
