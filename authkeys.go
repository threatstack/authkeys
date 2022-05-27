// authkeys - lookup a user's SSH keys as stored in LDAP
// authkeys.go: the whole thing.
//
// Copyright 2017-2022 F5 Inc.
// Licensed under the BSD 3-clause license; see LICENSE for more information.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"gopkg.in/ldap.v2"
)

type AuthkeysConfig struct {
	BaseDN        string
	DialTimeout   int
	KeyAttribute  string
	LDAPServer    string
	LDAPPort      int
	RootCAFile    string
	UserAttribute string
	BindDN        string
	BindPW        string
}

func NewConfig(fname string) AuthkeysConfig {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		panic(err)
	}
	config := AuthkeysConfig{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	return config
}

func main() {
	var config AuthkeysConfig
	var configfile string

	// Get configuration
	if os.Getenv("AUTHKEYS_CONFIG") == "" {
		configfile = "/etc/authkeys.json"
	} else {
		configfile = os.Getenv("AUTHKEYS_CONFIG")
	}
	if _, err := os.Stat(configfile); err == nil {
		config = NewConfig(configfile)
	}

	// Username should be our only attribute
	if len(os.Args) != 2 {
		log.Fatalf("Not enough parameters specified (or too many): just need LDAP username.")
	}
	username := os.Args[1]

	// Begin initial LDAP TCP connection. The LDAP library does have a Dial
	// function that does most of what we need -- but its default timeout is 60
	// seconds, which can be annoying if we're testing something in, say, Vagrant
	var conntimeout time.Duration
	if config.DialTimeout != 0 {
		conntimeout = time.Duration(config.DialTimeout) * time.Second
	} else {
		conntimeout = time.Duration(5) * time.Second
	}
	server, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", config.LDAPServer, config.LDAPPort),
		conntimeout)
	if err != nil {
		log.Fatal(err)
	}
	l := ldap.NewConn(server, false)
	l.Start()
	defer l.Close()

	// Need a place to store TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         config.LDAPServer,
	}

	// Configure additional trust roots if necessary
	if config.RootCAFile != "" {
		rootCerts := x509.NewCertPool()
		rootCAFile, err := ioutil.ReadFile(config.RootCAFile)
		if err != nil {
			log.Fatalf("Unable to read RootCAFile: %s", err)
		}
		if !rootCerts.AppendCertsFromPEM(rootCAFile) {
			log.Fatalf("Unable to append to CertPool from RootCAFile")
		}
		tlsConfig.RootCAs = rootCerts
	}

	// TLS our connection up
	err = l.StartTLS(tlsConfig)
	if err != nil {
		log.Fatalf("Unable to start TLS connection: %s", err)
	}

	// If we have a BindDN go ahead and bind before searching
	if config.BindDN != "" && config.BindPW != "" {
		err := l.Bind(config.BindDN, config.BindPW)
		if err != nil {
			log.Fatalf("Unable to bind: %s", err)
		}
	}

	// Set up an LDAP search and actually do the search
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(%s=%s)", config.UserAttribute, username),
		[]string{config.KeyAttribute},
		nil,
	)
	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	if len(sr.Entries) == 0 {
		log.Fatalf("No entries returned from LDAP")
	} else if len(sr.Entries) > 1 {
		log.Fatalf("Too many entries returned from LDAP")
	}

	// Get the keys & print 'em. This will only print keys for the first user
	// returned from LDAP, but if you have multiple users with the same name maybe
	// setting a different BaseDN may be useful.
	keys := sr.Entries[0].GetAttributeValues(config.KeyAttribute)
	for _, key := range keys {
		fmt.Printf("%s\n", key)
	}
}
