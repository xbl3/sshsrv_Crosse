package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	log "bitbucket.org/crosse3/gosimplelogger"
)

const (
	serviceName = "ssh"
	defaultPort = 22
)

var sshPath string

// GetSSHEndpoint tries to determine how to connect to a particular host
// via SSH.  GetSSHEndpoint first attempts to discover the endpoint via
// DNS SRV records of the form "_ssh._tcp.<hostname>".  If found,
// GetSSHEndpoint will return the target host and port to connect to
// instead of the "bare" DNS hostname.  If no SRV record is found, it
// will simply return the hostname and default SSH port (22).
func GetSSHEndpoint(hostname string) (target string, port uint16, err error) {
	cname, srvAddrs, err := net.LookupSRV(serviceName, "tcp", hostname)
	if err != nil {
		if _, ok := err.(*net.DNSError); ok {
			// DNS-related error.
			log.Verbosef("error: %v", err)
			err = nil
		} else {
			// Non-DNS error.  Probably want to stop now.
			log.Fatal(err)
		}
	} else {
		log.Verbosef("Retrieved record for %v", cname)
	}

	if len(srvAddrs) > 0 {
		log.Verbosef("Found %d SRV record(s)", len(srvAddrs))

		for i, r := range srvAddrs {
			log.Verbosef("Record %d:\t%d %d %d %s", i, r.Priority, r.Weight, r.Port, r.Target)
		}

		// "The returned records are sorted by priority and randomized
		// by weight within a priority", so return details for the first
		// one in the list.

		// The target DNS names are fully-specified with the root ("."),
		// so trim that off.
		target = strings.TrimRight(srvAddrs[0].Target, ".")
		port = srvAddrs[0].Port
	} else {
		log.Verbosef("No SRV record found for %v", hostname)
		target = hostname
		port = defaultPort
	}

	// TODO: Extend this to return the entire list in priority-order
	return
}

func init() {
	var err error
	sshPath, err = exec.LookPath("ssh")
	if err != nil {
		log.Fatal("Could not find ssh!")
	}

	var verbose = flag.Bool("v", false, "enable verbose logging")
	flag.Parse()

	if *verbose {
		log.LogLevel = log.LogVerbose
	}
}

func main() {
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	host := flag.Args()[0]
	sshArgs := flag.Args()[1:]
	targetHost, targetPort, err := GetSSHEndpoint(host)
	if err != nil {
		log.Fatal(err)
	}

	log.Verbosef("Target for %v is %v:%v", host, targetHost, targetPort)


	args := []string{}
	args = append(args, fmt.Sprintf("-p %d", targetPort))
	args = append(args, targetHost)
	args = append(args, sshArgs...)
	cmd := exec.Command(sshPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Verbosef("Connecting to %v:%v", targetHost, targetPort)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	cmd.Wait()
}
