// authkeys - lookup a user's SSH keys as stored in LDAP
// authkeys.go: the whole thing.
//
// Copyright 2017 Threat Stack, Inc. See LICENSE for license information.
// Author: Patrick T. Cable II <pat.cable@threatstack.com>

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"gopkg.in/ldap.v2"
	"io/ioutil"
	"log"
	"os"
)

type AuthkeysConfig struct {
	BaseDN string
	KeyAttribute string
	LDAPServer string
	LDAPPort int
	RootCAFile string
	UserAttribute string
}

func NewConfig(fname string) AuthkeysConfig {
	data,err := ioutil.ReadFile(fname)
	if err != nil{
		panic(err)
	}
	config := AuthkeysConfig{}
	err = json.Unmarshal(data,&config)
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

	// Parse arguments. Old versions of authkeys took 3 arguments, the only
	// relevent one today is the username so this is a workaround to make
	// deployment easier for the company that wrote it :)
	if len(os.Args) != 2 {
		log.Fatalf("Not enough parameters specified: Need LDAP username.")
	}
	username := os.Args[1]

	// Begin initial LDAP TCP connection
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.LDAPServer, config.LDAPPort))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// Need a place to store TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName: config.LDAPServer,
	}

	// Configure additional trust roots
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

	// Get the keys & print 'em. This will only print keys for the first user
	// returned from LDAP, but if you have multiple users with the same name maybe
	// setting a different BaseDN may be useful.
	keys := sr.Entries[0].GetAttributeValues(config.KeyAttribute)
	for _, key := range keys {
		fmt.Printf("%s\n", key)
	}
}
