# Using SSL Between Draft and Draftd

This document explains how to create strong SSL/TLS connections between the
Draft client (Draft) and server (Draftd). The emphasis here is on creating an
internal CA, and using both the cryptographic and identity functions of SSL.

Configuring SSL is considered an advanced topic, and knowledge of Draft is assumed.

## Overview

The Draftd authentication model uses client-side SSL certificates. Draftd itself
verifies these certificates using a certificate authority. Likewise, the client
also verifies Draftd's identity by certificate authority.

NOTE: For now this document concerns configuring the Draft client and server to
communicate using SSL certificates. Specifically, TLS is used to secure gRPC
communication between the Draft client and server.

There are numerous possible configurations for setting up certificates and authorities,
but the method we cover here will work for most situations.

In this guide, we will show how to:

- Create a private CA that is used to issue certificates for Draftd clients and
  servers.
- Create a certificate for Draftd
- Create a certificate for the Draft client
- Create a Draftd instance that uses the certificate
- Configure the Draft client to use the CA and client-side certificate

By the end of this guide, you should have a Draftd instance running that will
only accept connections from clients who can be authenticated by SSL certificate.

## Generating Certificate Authorities and Certificates

One way to generate SSL CAs is via the `openssl` command line tool. There are many
guides and best practices documents available online. This explanation is focused
on getting ready within a small amount of time. For production configurations,
we urge readers to read [the official documentation](https://www.openssl.org) and
consult other resources.

### Generate a Certificate Authority

The simplest way to generate a certificate authority is to run two commands:

```console
$ openssl genrsa -out ./ca.key.pem 4096
$ openssl req -key ca.key.pem -new -x509 -days 7300 -sha256 -out ca.cert.pem -extensions v3_ca
Enter pass phrase for ca.key.pem:
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:US
State or Province Name (full name) [Some-State]:CO
Locality Name (eg, city) []:Boulder
Organization Name (eg, company) [Internet Widgits Pty Ltd]:draft
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:draft
Email Address []:draftd@example.com
```

Note that the data input above is _sample data_. You should customize to your own
specifications.

The above will generate both a secret key and a CA. Note that these two files are
very important. The key in particular should be handled with particular care.

Often, you will want to generate an intermediate signing key. For the sake of brevity,
we will be signing keys with our root CA.

### Generating Certificates

We will be generating two certificates, each representing a type of certificate:

- One certificate is for Draftd. You will want one of these _per draftd host_ that
  you run.
- One certificate is for the user. You will want one of these _per draft client_.

Since the commands to generate these are the same, we'll be creating both at the
same time. The names will indicate their target.

First, the Draftd key:

```console
$ openssl genrsa -out ./draftd.key.pem 4096
Generating RSA private key, 4096 bit long modulus
..........................................................................................................................................................................................................................................................................................................................++
............................................................................++
e is 65537 (0x10001)
Enter pass phrase for ./draftd.key.pem:
Verifying - Enter pass phrase for ./draftd.key.pem:
```

Next, generate the Draft client's key:

```console
$ openssl genrsa -out ./draft.key.pem 4096
Generating RSA private key, 4096 bit long modulus
.....++
......................................................................................................................................................................................++
e is 65537 (0x10001)
Enter pass phrase for ./draft.key.pem:
Verifying - Enter pass phrase for ./draft.key.pem:
```

Again, for production use you will generate one client certificate for each user.

Next we need to create certificates from these keys. For each certificate, this is
a two-step process of creating a CSR, and then creating the certificate.

```console
$ openssl req -key draftd.key.pem -new -sha256 -out draftd.csr.pem
Enter pass phrase for draftd.key.pem:
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:US
State or Province Name (full name) [Some-State]:CO
Locality Name (eg, city) []:Boulder
Organization Name (eg, company) [Internet Widgits Pty Ltd]:Draftd Server
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:draftd-server
Email Address []:

Please enter the following 'extra' attributes
to be sent with your certificate request
A challenge password []:
An optional company name []:
```

And we repeat this step for the Draft client certificate:

```console
$ openssl req -key draft.key.pem -new -sha256 -out draft.csr.pem
# Answer the questions with your client user's info
```

(In rare cases, we've had to add the `-nodes` flag when generating the request.)

Now we sign each of these CSRs with the CA certificate we created:

```console
$ openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in draftd.csr.pem -out draftd.cert.pem
Signature ok
subject=/C=US/ST=CO/L=Boulder/O=Draftd Server/CN=draftd-server
Getting CA Private Key
Enter pass phrase for ca.key.pem:
```

And again for the client certificate:

```console
$ openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in draft.csr.pem -out draft.cert.pem
```

At this point, the important files for us are these:

```
# The CA. Make sure the key is kept secret.
ca.cert.pem
ca.key.pem
# The Draft client files
draft.cert.pem
draft.key.pem
# The Draftd server files.
draftd.cert.pem
draftd.key.pem
```

Now we're ready to move on to the next steps.

## Creating a Custom Draftd Installation

The Draft client includes full support for creating a deployment configured for SSL.
By specifying a few flags, the `draft init` command can create a new Draftd installation
complete with all of our SSL configuration.

To take a look at what this will generate, run this command:

```console
$ draft init --dry-run --debug --draftd-tls --draftd-tls-cert ./draftd.cert.pem --draftd-tls-key ./draftd.key.pem --draftd-tls-verify --tls-ca-cert ca.cert.pem
```

The output will show you the values file used to install the Draft chart. Your SSL
information will be preloaded into the Secret, which the Deployment will mount to
pods as they start up.


## Configuring the Draft Client

The Draftd server is now running with TLS protection. It's time to configure the
Draft client to enable TLS-secured operations.

For a quick test, we can specify our configuration manually.

```console
draft up --tls --tls-ca-cert ca.cert.pem --tls-cert draft.cert.pem --tls-key draft.key.pem
```

This configuration sends our client-side certificate to establish identity, uses
the client key for encryption, and uses the CA certificate to validate the remote
Draftd's identity.

Typing a line that that is cumbersome, though. The shortcut is to move the key,
cert, and CA into `$DRAFT_HOME`:

```console
$ cp ca.cert.pem $(draft home)/ca.pem
$ cp draft.cert.pem $(draft home)/cert.pem
$ cp draft.key.pem $(draft home)/key.pem
```

With this, you can simply run `draft up --tls` to enable TLS.

### Troubleshooting

*Running a command, I get `Error: transport is closing`*

This is almost always due to a configuration error in which the client is missing
a certificate (`--tls-cert`) or the certificate is bad.

*I'm using a certificate, but get `Error: remote error: tls: bad certificate`*

This means that Draftd's CA cannot verify your certificate. In the examples above,
we used a single CA to generate both the client and server certificates. In these
examples, the CA has _signed_ the client's certificate. We then load that CA
up to Draftd. So when the client certificate is sent to the server, Draftd
checks the client certificate against the CA.

*If I use `--tls-verify` on the client, I get `Error: x509: certificate is valid for draftd-server, not localhost`*

If you plan to use `--tls-verify` on the client, you will need to make sure that
the host name that Draft connects to matches the host name on the certificate. In
some cases this is awkward, since Draft will connect over localhost, or the FQDN is
not available for public resolution.

## References

https://github.com/denji/golang-tls
https://www.openssl.org/docs/
https://jamielinux.com/docs/openssl-certificate-authority/sign-server-and-client-certificates.html
