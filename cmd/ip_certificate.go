package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/service"
)

// runIpCert handles `sui ip-cert <issue|renew|status|disable>`. Issuance happens
// in-process via go-acme/lego (the panel binary embeds it, keeping the zero
// os/exec invariant); the management menu (s-ui.sh) stops the panel first to
// free the HTTP-01 challenge port and restarts it afterwards so the panel's web
// listener reloads the freshly issued certificate. Returns a process exit code.
func runIpCert(args []string) int {
	if len(args) == 0 {
		printIpCertUsage()
		return 2
	}
	switch args[0] {
	case "issue":
		return ipCertIssue(args[1:])
	case "renew":
		return ipCertRenew(args[1:])
	case "status":
		return ipCertStatus()
	case "disable":
		return ipCertDisable()
	default:
		fmt.Println("ip-cert: unknown sub-command:", args[0])
		printIpCertUsage()
		return 2
	}
}

func printIpCertUsage() {
	fmt.Println("Usage:")
	fmt.Println("    sui ip-cert issue -ip <ip> -email <email> [-port 80] [-no-renew]")
	fmt.Println("        issue a Let's Encrypt certificate for a bare IP and apply it to the panel HTTPS listener")
	fmt.Println("    sui ip-cert renew")
	fmt.Println("        re-issue now using the stored IP/email/port")
	fmt.Println("    sui ip-cert status")
	fmt.Println("        show the managed certificate state")
	fmt.Println("    sui ip-cert disable")
	fmt.Println("        turn off 12h auto-renewal")
}

// newIpCertService opens the database and returns an IP-certificate service with
// no live runtime — the CLI never restarts the panel itself.
func newIpCertService() (*service.IpCertificateService, error) {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		return nil, err
	}
	return &service.IpCertificateService{Settings: &service.SettingService{}}, nil
}

func ipCertIssue(args []string) int {
	fs := flag.NewFlagSet("ip-cert issue", flag.ContinueOnError)
	var ip, email string
	var port int
	var noRenew bool
	fs.StringVar(&ip, "ip", "", "public IP address to certify")
	fs.StringVar(&email, "email", "", "ACME account email")
	fs.IntVar(&port, "port", 80, "HTTP-01 challenge port")
	fs.BoolVar(&noRenew, "no-renew", false, "do not enable 12h auto-renewal")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ip = strings.TrimSpace(ip)
	if ip == "" {
		fmt.Println("ip-cert: -ip is required")
		printIpCertUsage()
		return 2
	}

	svc, err := newIpCertService()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}

	fmt.Printf("ip-cert: issuing certificate for %s (HTTP-01 challenge on port %d)...\n", ip, port)
	status, err := svc.IssueForCLI(context.Background(), ip, strings.TrimSpace(email), port)
	if err != nil {
		fmt.Println("ip-cert: issue failed:", err)
		return 1
	}

	if !noRenew {
		if err := svc.Settings.SetIpCertEnabled(true); err != nil {
			fmt.Println("ip-cert: warning: could not enable auto-renew:", err)
		} else {
			status.Enabled = true
		}
	}

	printIpCertStatus(status)
	fmt.Println("ip-cert: applied to the panel HTTPS listener; restart the panel to load it.")
	return 0
}

func ipCertRenew(args []string) int {
	fs := flag.NewFlagSet("ip-cert renew", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	svc, err := newIpCertService()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}

	ip, err := svc.Settings.GetIpCertTargetIP()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		fmt.Println("ip-cert: no stored IP; run `sui ip-cert issue` first")
		return 1
	}
	email, _ := svc.Settings.GetIpCertEmail()
	port, _ := svc.Settings.GetIpCertChallengePort()

	fmt.Printf("ip-cert: re-issuing certificate for %s...\n", ip)
	status, err := svc.IssueForCLI(context.Background(), ip, strings.TrimSpace(email), port)
	if err != nil {
		fmt.Println("ip-cert: renew failed:", err)
		return 1
	}

	printIpCertStatus(status)
	fmt.Println("ip-cert: applied to the panel HTTPS listener; restart the panel to load it.")
	return 0
}

func ipCertStatus() int {
	svc, err := newIpCertService()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}
	status, err := svc.GetStatus()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}
	printIpCertStatus(status)
	return 0
}

func ipCertDisable() int {
	svc, err := newIpCertService()
	if err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}
	if err := svc.Settings.SetIpCertEnabled(false); err != nil {
		fmt.Println("ip-cert:", err)
		return 1
	}
	fmt.Println("ip-cert: auto-renewal disabled")
	return 0
}

func printIpCertStatus(s service.IpCertStatus) {
	fmt.Println("IP certificate status:")
	fmt.Println("\tAuto-renew:\t", s.Enabled)
	if s.TargetIP != "" {
		fmt.Println("\tTarget IP:\t", s.TargetIP)
	}
	fmt.Println("\tIssued:   \t", s.Issued)
	if s.Issued {
		fmt.Println("\tExpires:  \t", s.NotAfter)
		fmt.Printf("\tDays left:\t %.1f\n", s.DaysRemaining)
	}
	if s.LastIssue != "" {
		fmt.Println("\tLast issue:\t", s.LastIssue)
	}
	if s.CertPath != "" {
		fmt.Println("\tCert path:\t", s.CertPath)
	}
}
