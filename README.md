# authkeys

`authkeys` is a tool written in Go that you can use with OpenSSH as an
`AuthorizedKeysCommand`. It'll reach out to LDAP and get keys and display them
on stdout.

## Pre-Requisites

You'll need an LDAP server that has a
[schema](http://pig.made-it.com/ldap-openssh.html) installed for storing SSH
keys as part of an entry. Also, your LDAP server will need to use STARTTLS over
port 389, as opposed to LDAPS.

## Installation

To build a binary, you can use `go get`:
`go get -d github.com/threatstack/authkeys`.

You'll need to put that binary somewhere (we use `/usr/sbin` because
we make a package for it using [fpm](https://github.com/jordansissel/fpm))
and make sure the binary is chmod'ed to `0555`.

Then, add to your `sshd_config`:

```
AuthorizedKeysCommandUser nobody
AuthorizedKeysCommand /usr/sbin/authkeys
```

Now when you log in, OpenSSH will run (in this example) `/usr/sbin/authkeys`
with the username as the first argument. Authkeys will return the keys from
LDAP, and the user should be logged in if there is a match. Of note: OpenSSH
will prefer a local `~/.ssh/authorized_keys` file over keys returned from
`AuthorizedKeysCommand`, so make sure you test with a user that _doesn't_ have
that file.

At Threat Stack, we use Chef to deploy our authkeys package as part of our LDAP client
setup -- just using a template and package resource. We leverage the OpenSSH
cookbook using a `node.override` for the `authorized_keys_command` and
`authorized_keys_command_user` variables.

## Configuration

Authkeys is configured using a JSON file. By default, it'll look in
`/etc/authkeys.json` but you can override this with the `AUTHKEYS_CONFIG`
environment variable for testing.

```
{
  "BaseDN": "",
  "DialTimeout": 5,
  "KeyAttribute": "",
  "LDAPServer": "",
  "LDAPPort": 389,
  "RootCAFile": "",
  "UserAttribute": "",
}
```

| Variable        | Type   | Purpose                                              | Possible Value                       |
|-----------------|--------|------------------------------------------------------|--------------------------------------|
| `BaseDN`        | String | Base DN for your LDAP server                         | `dc=spiffy,dc=io`                    |
| `DialTimeout`   | Int    | A connection timeout if LDAP isnt reachable [Note 1] | `5`                                  |
| `KeyAttribute`  | String | LDAP Attribute for the SSH key                       | `sshPublicKey`                       |
| `LDAPServer`    | String | Hostname of your LDAP server                         | `ldap.spiffy.io`                     |
| `LDAPPort`      | Int    | Port to talk to LDAP on                              | `389`                                |
| `RootCAFile`    | String | A path to a file full of trusted root CAs [Note 2]   | `/etc/ssl/certs/ca-certificates.crt` |
| `UserAttribute` | String | LDAP Attribute for a User                            | `uid`                                |

### Notes

1. Defaults to 5 seconds
1. If blank, Go will attempt to use system trust roots.

## Usage

`authkeys [username]` will look up the user in LDAP and get their keys. Simple
as that.

## Changelog
If you're wondering why this started at version 2.0.0, it's because we've been
using this tool internally for a while, and we cleaned it up for external
consumption :)

Version 2.1 added a quicker TCP timeout. You can set this using the
`DialTimeout` attribute.

Version 2.1.1 adds in some kinder error handling for when LDAP returns nothing
(or when it returns >1 result)

## Contribution

1. Fork
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Create a new Pull Request

## Author

Patrick Cable (@patcable)
