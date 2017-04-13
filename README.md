# authkeys

`authkeys` is a tool written in Go that you can use with OpenSSH as an
`AuthorizedKeysCommand`. It'll reach out to LDAP and get keys and display them
on stdout.

## Installation

To install, use `go get`: `go get -d github.com/threatstack/authkeys`. You'll
need to put this in a directory and chmod it to `0555` (or if you want to be
most paranoid, `0500`)

## Pre-Requisites

You'll need an LDAP server that has a
[schema](http://pig.made-it.com/ldap-openssh.html) installed for storing SSH
keys as part of a record.

Also, your LDAP server will need to use STARTTLS.

## Configuration

Authkeys is configured using a JSON file -- by default, it'll look in
`/etc/authkeys.json` but you can override this with the `AUTHKEYS_CONFIG`
environment variable for testing.

```
{
  "BaseDN": "",
  "KeyAttribute": "",
  "LDAPServer": "",
  "LDAPPort": 389,
  "RootCAFile": "",
  "UserAttribute": "",
}
```

| Variable        | Type   | Purpose                                                | Possible Value                       |
|-----------------|--------|--------------------------------------------------------|--------------------------------------|
| `BaseDN`        | String | Base DN for your LDAP server                           | `dc=spiffy,dc=io`                    |
| `KeyAttribute`  | String | LDAP Attribute for the SSH key                         | `sshPublicKey`                       |
| `LDAPServer`    | String | Hostname of your LDAP server                           | `ldap.spiffy.io`                     |
| `LDAPPort`      | Int    | Port to talk to LDAP on                                | `389`                                |
| `RootCAFile`    | String | A path to a file full of trusted root CAs [See note 1] | `/etc/ssl/certs/ca-certificates.crt` |
| `UserAttribute` | String | LDAP Attribute for a User                              | `uid`                                |

### Notes

1. If blank, Go will attempt to use system trust roots.

## Usage

`authkeys [username]` will look up the user in LDAP and get their keys. Simple
as that.

## History
If you're wondering why this is version 2.0.0, it's because we've been using
this tool internally for a while, and we cleaned it up for external consumption
:)

## Contribution

1. Fork
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Create a new Pull Request

## Author

Patrick Cable (@patcable)
