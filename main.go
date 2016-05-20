package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"strings"
	"time"
)

const Version = "0.0.0"
const DefaultUpstream = "8000"

var host = flag.String("host", "localhost", "")
var listen = flag.String("listen", "127.0.0.1:8443", "")

func usage() {
	prog := os.Args[0]
	fmt.Fprintf(os.Stderr, "usage: %s [-host=<hostname>] [-listen=<address>] [upstream]\n", prog)
	fmt.Fprintf(os.Stderr, "version: %s\n", Version)
	fmt.Fprint(os.Stderr, "examples:\n")
	fmt.Fprintf(os.Stderr, "    %s 8000                # create proxy to localhost:8000\n", prog)
	fmt.Fprintf(os.Stderr, "    %s -host=foo.dev 9000  # generate a cert for foo.dev:9000\n", prog)
	fmt.Fprintf(os.Stderr, "    %s -listen=:8888       # listen on port 8888\n", prog)
}

func usageAndExit(s string) {
	fmt.Fprintf(os.Stderr, "%s\n", s)
	flag.CommandLine.Usage()
	os.Exit(1)
}

func init() {
	flag.CommandLine.Usage = usage
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func generateCert(host string) (string, string) {
	var err error

	dir := "/tmp/https-certs/" + host + "/"
	certPath := dir + "cert.pem"
	keyPath := dir + "key.pem"

	if exists(certPath) && exists(keyPath) {
		return certPath, keyPath
	}

	log.Println("Generating new certificates...")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		IsCA: true,

		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		log.Fatalf("Failed to write certificates: %s", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("failed to open key.pem for writing: %s", err)
	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	return certPath, keyPath
}

func reverseProxy(host string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = host
		req.Header.Set("X-Forwarded-Proto", "https")
	}
	return &httputil.ReverseProxy{Director: director}
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) > 1 {
		usageAndExit("too many args")
	}

	if len(args) == 0 {
		args = append(args, DefaultUpstream)
	}

	listenAddr, err := net.ResolveTCPAddr("tcp", *listen)
	if err != nil {
		usageAndExit("failed to parse -listen")
	}

	upstream := args[0]
	if !strings.Contains(upstream, ":") {
		upstream = "127.0.0.1:" + upstream
	}

	cert, key := generateCert(*host)
	proxy := reverseProxy(upstream)

	fmt.Print(` _     _   _
| |   | | | |
| |__ | |_| |_ _ __  ___
| '_ \| __| __| '_ \/ __|
| | | | |_| |_| |_) \__ \
|_| |_|\__|\__| .__/|___/
              | |
              |_|

`)

	fmt.Println("upstream: " + upstream)
	fmt.Println("listen:   " + *listen)
	if listenAddr.Port == 443 {
		fmt.Println(fmt.Sprintf("proxy:    https://%s/", *host))
	} else {
		fmt.Println(fmt.Sprintf("proxy:    https://%s:%d/", *host, listenAddr.Port))
	}
	fmt.Println("")
	fmt.Println("==> let's go!")
	log.Fatal(http.ListenAndServeTLS(*listen, cert, key, proxy))
}
