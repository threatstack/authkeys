// authkeys - lookup a user's SSH keys as stored in LDAP
// authkeys.go: the whole thing.
//
// Copyright 2017 Threat Stack, Inc.
// Licensed under the BSD 3-clause license; see LICENSE for more information.
// Author: Patrick T. Cable II <pat.cable@threatstack.com>

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
	"strings"
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

type User struct {
	Uid		string `json:"id"`
	UidNumber   	string `json:"uid"`
	MemberOf    	[]string `json:"groups"`
	HomeDirectory  	string `json:"home"`
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

	groupPtr := flag.String("group", "", "List members of this JumpCloud LDAP group")
	flag.Parse()
	listUsers := false
	username := ""
	if *groupPtr != "" {
		listUsers = true
	} else if len(os.Args) != 2 {
		log.Fatalf("Not enough parameters specified (or too many): just need LDAP username.")
	} else {
		username = os.Args[1]
	}

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
		err = l.Bind(config.BindDN, config.BindPW)
		if err != nil {
			log.Fatalf("Unable to bind: %s", err)
		}
	}

	var searchRequest *ldap.SearchRequest
	if listUsers {
		searchRequest = ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(&(objectClass=inetOrgPerson)(memberOf=cn=%s,%s))", *groupPtr, config.BaseDN),
			[]string{"uid","uidNumber","memberOf","homeDirectory"}, // attributes to retrieve
			nil,
		)
	} else {
		// Set up an LDAP search and actually do the search
		searchRequest = ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(%s=%s)", config.UserAttribute, username),
			[]string{config.KeyAttribute},
			nil,
		)
	}

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	if len(sr.Entries) == 0 {
		log.Fatalf("No entries returned from LDAP")
	} else if !listUsers && (len(sr.Entries) > 1) {
		log.Fatalf("Too many entries returned from LDAP")
	}

	var attribute string
	cn := "cn="
	if listUsers {
	var Users []User
		for _, entry := range sr.Entries {
			rawMemberOf := entry.GetAttributeValues("memberOf")
			var memberOf []string
			for group := range(rawMemberOf) {
			        cnLoc := strings.Index(rawMemberOf[group], cn)
				termLoc := strings.Index(rawMemberOf[group], ",")
				memberOf = append(memberOf, rawMemberOf[group][cnLoc+len(cn):termLoc])
			}
			Users = append(Users,User{
				Uid: string(entry.GetAttributeValue("uid")),
				UidNumber: string(entry.GetAttributeValue("uidNumber")),
				MemberOf: memberOf,
				HomeDirectory: string(entry.GetAttributeValue("homeDirectory")),
			})
		}
		myUsers, err := json.Marshal(Users)
		if err != nil {
			log.Fatal(err)
		}
	        fmt.Printf("%s\n", myUsers)
	} else {
		attribute = config.KeyAttribute
		for _, entry := range sr.Entries {
			keys := entry.GetAttributeValues(attribute)
			for _, key := range keys {
				fmt.Printf("%s\n", key)
		        }
		}
	}
	// Get the keys & print 'em. This will only print keys for the first user
	// returned from LDAP, but if you have multiple users with the same name maybe
	// setting a different BaseDN may be useful.
}
