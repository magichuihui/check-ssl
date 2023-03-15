// Copyright 2022 kyra. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"regexp"
	"sync"
	"time"
)

const defaultConcurrency = 8

const (
	errExpiringShortly = "%s: ** '%s' (S/N %X) expires in %d hours! **"
	errExpiringSoon    = "%s: '%s' (S/N %X) expires in roughly %d days"
	errSunsetAlg       = "%s: '%s' (S/N %X) expires after the sunset date for its signature algorithm '%s'"
)

type sigAlgSunset struct {
	name      string    // Human readable name of signature algorithm
	sunsetsAt time.Time // Time the algorithm will be sunset
}

// sunsetSigAlgs is an algorithm to string mapping for signature algorithms
// which have been or are being deprecated.  See the following links to learn
// more about SHA1's inclusion on this list.
//
// - https://technet.microsoft.com/en-us/library/security/2880823.aspx
// - http://googleonlinesecurity.blogspot.com/2014/09/gradually-sunsetting-sha-1.html
var sunsetSigAlgs = map[x509.SignatureAlgorithm]sigAlgSunset{
	x509.MD2WithRSA: {
		name:      "MD2 with RSA",
		sunsetsAt: time.Now(),
	},
	x509.MD5WithRSA: {
		name:      "MD5 with RSA",
		sunsetsAt: time.Now(),
	},
	x509.SHA1WithRSA: {
		name:      "SHA1 with RSA",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	x509.DSAWithSHA1: {
		name:      "DSA with SHA1",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	x509.ECDSAWithSHA1: {
		name:      "ECDSA with SHA1",
		sunsetsAt: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

type certErrors struct {
	commonName string
	errs       []error
}

type hostResult struct {
	host  string
	err   error
	certs []certErrors
}

type SSLChecker struct {
	Records []string `json:"Records"`
	hosts   <-chan string
}

func (checker *SSLChecker) ProcessHosts() (errMessage string) {

	done := make(chan struct{})
	defer close(done)

	checker.queueHosts(done)
	results := make(chan hostResult)

	var wg sync.WaitGroup
	wg.Add(defaultConcurrency)
	for i := 0; i < defaultConcurrency; i++ {
		go func() {
			checker.processQueue(done, results)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	re := regexp.MustCompile(`(?i)Certifi`)
	for r := range results {
		if r.err != nil {
			match := re.MatchString(r.err.Error())
			if match {
				log.Printf("%s: %v\n", r.host, r.err)
			}
			continue
		}
		for _, cert := range r.certs {
			for _, err := range cert.errs {
				errMessage += r.host + ": " + err.Error() + "\n"
			}
		}
	}

	return errMessage
}

func (checker *SSLChecker) AppendHosts(hosts []string) {
	checker.Records = append(checker.Records, hosts...)
}

func (checker *SSLChecker) Output() {
	fmt.Println(checker.Records)
}

func (checker *SSLChecker) queueHosts(done <-chan struct{}) {
	hosts := make(chan string)
	go func() {
		defer close(hosts)

		for _, host := range checker.Records {
			if len(host) == 0 || host[0] == '#' {
				continue
			}
			select {
			case hosts <- host:
			case <-done:
				return
			}
		}
	}()

	checker.hosts = hosts
}

func (checker *SSLChecker) processQueue(done <-chan struct{}, results chan<- hostResult) {
	for host := range checker.hosts {
		select {
		case results <- checker.checkHost(host):
		case <-done:
			return
		}
	}
}

func (checker *SSLChecker) checkHost(host string) (result hostResult) {
	result = hostResult{
		host:  host,
		certs: []certErrors{},
	}

	conf := &tls.Config{}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", host, conf)

	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	timeNow := time.Now()
	checkedCerts := make(map[string]struct{})
	for _, chain := range conn.ConnectionState().VerifiedChains {
		for certNum, cert := range chain {
			if _, checked := checkedCerts[string(cert.Signature)]; checked {
				continue
			}
			checkedCerts[string(cert.Signature)] = struct{}{}
			cErrs := []error{}

			// Check the expiration.
			if timeNow.AddDate(0, 0, 30).After(cert.NotAfter) {
				expiresIn := int64(cert.NotAfter.Sub(timeNow).Hours())
				if expiresIn <= 48 {
					cErrs = append(cErrs, fmt.Errorf(errExpiringShortly, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn))
				} else {
					cErrs = append(cErrs, fmt.Errorf(errExpiringSoon, host, cert.Subject.CommonName, cert.SerialNumber, expiresIn/24))
				}
			}

			// Check the signature algorithm, ignoring the root certificate.
			if alg, exists := sunsetSigAlgs[cert.SignatureAlgorithm]; exists && certNum != len(chain)-1 {
				if cert.NotAfter.Equal(alg.sunsetsAt) || cert.NotAfter.After(alg.sunsetsAt) {
					cErrs = append(cErrs, fmt.Errorf(errSunsetAlg, host, cert.Subject.CommonName, cert.SerialNumber, alg.name))
				}
			}

			result.certs = append(result.certs, certErrors{
				commonName: cert.Subject.CommonName,
				errs:       cErrs,
			})
		}
	}

	return
}
