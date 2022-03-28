# Vulcan Checks
This repository holds the code for each Vulcan check's main binary.

## Vendoring
Currently there's no vendoring provided for this project.

## Current list of [Checks](https://github.com/adevinta/vulcan-checks/tree/master/cmd)
* **vulcan-aws-alerts** - Warns about CA issues in AWS RDS
* **vulcan-aws-trusted-advisor** - Checks AWS Trusted Advisor for security findings
* **vulcan-burp** - Runs a PortSwigger [Burp Enterprise](https://portswigger.net/burp/enterprise) scan
* **vulcan-certinfo** - Extracts information about SSL/TLS certificates
* **vulcan-dkim** - Checks if a domain (asset with a SOA record) have valid DNS configuration for DKIM
* **vulcan-dmarc** - Checks if a domain (asset with a SOA record) have valid DNS configuration for DMARC
* **vulcan-drupal** - Checks for vulnerabilities in Drupal CMS
* **vulcan-exposed-amt** - Checks if an asset has the Intel AMT port exposed and whether is it vulnerable or not
* **vulcan-exposed-bgp** - Checks for exposed BGP port on Internet routers
* **vulcan-exposed-db** - Checks if an asset has open database well known ports
* **vulcan-exposed-files** - Check asset for sensitive files exposed on HTTP server
* **vulcan-exposed-ftp** - Checks if an asset has open FTP well known ports and if they allow anonymous logins or vulnerable to bounce attack
* **vulcan-exposed-hdfs** - Checks if an EMR cluster is exposed to the Internet
* **vulcan-exposed-http** - Checks if an asset has open HTTP well known ports
* **vulcan-exposed-http-resources** - Checks if a web address exposes sensitive resources
* **vulcan-exposed-memcached** - Checks if an asset has exposed a memcached server
* **vulcan-exposed-rdp** - Checks if a Microsoft Remote Desktop service is exposed to the Internet
* **vulcan-exposed-router-ports** - Checks if an asset has open router well known ports
* **vulcan-exposed-services** - Checks if a host has any port opened by scanning the 1000 most common TCP and UDP ports
* **vulcan-exposed-ssh** - Checks SSH server configuration for compliance with Mozilla OpenSSH guidelines
* **vulcan-exposed-varnish** - Checks if an asset is a Web Cache, and also if it is a Varnish
* **vulcan-github-alerts** - Retrieves existing vulnerability alerts for a Github repository
* **vulcan-gozuul** - Checks if a Zuul Gateway is vulnerable to Remote Code Execution as specified in nflx-2016-003
* **vulcan-heartbleed** - Checks if an asset is vulnerable to heartbleed vulnerability
* **vulcan-host-discovery** - Performs a quick Nmap ping scan that identifies which hosts are up
* **vulcan-http-headers** - Analyzes the security of a website based on its HTTP headers
* **vulcan-ipv6** - Checks for IPv6 presence
* **vulcan-lucky** - Checks if an TLS asset is vulnerable to LuckyMinus20 attack
* **vulcan-masscan** - Checks if a host has any port opened by scanning the whole TCP port range using masscan
* **vulcan-mx** - Looks for MX DNS Records on a given domain
* **vulcan-nessus** - Runs a Nessus scan
* **vulcan-prowler** - Checks compliance against CIS AWS Foundations Benchmark
* **vulcan-results-load-test** - Internal testing check, not for production
* **vulcan-retirejs** - Checks for vulnerabilities in JS frontend dependencies
* **vulcan-s3-takeover** - Checks for a vulnerability related to domain names pointing to a S3 static website when the S3 bucket has been deleted
* **vulcan-seekret** - Checks if a Git repository contains secrets like passwords, API tokens or private keys
* **vulcan-sleep** - Internal testing check, not for production
* **vulcan-smtp-open-relay** - Checks for exposed SMTP, and if they are open relay
* **vulcan-spf** - Checks if a domain (asset with a SOA record) have valid DNS configuration for SPF
* **vulcan-tls** - Analyzes TLS health of an asset
* **vulcan-trivy** - Checks if a Docker image uses vulnerable packages or dependencies using Trivy
* **vulcan-unclassified** - Example vulnerability to test the monitoring of unclassified vulnerabilities - not for production
* **vulcan-vulners** - Runs https://vulners.com/api/v3/burp/software/
* **vulcan-wpscan** - Checks Wordpress sites for vulnerabilities using the open source wpscan utility
* **vulcan-zap** - Checks for vulnerabilities in web applications using OWASP ZAP

## Building and testing

This project is primarily built using the [vulcan-checks-bsys](https://github.com/adevinta/vulcan-checks-bsys) project.

But it's possible to build each one of the checks with `go build`.

In every check directory there is an example configuration file called `local.toml.example`, most checks reads a
file named `local.toml` if you pass the `-t` parameter, so by copying that file you can do a local test of your code
before you commit.

The checks that are not self-contained, and instead rely on external binaries, can be locally tested with
`vulcan-checks-bsys`. By running `vulcan-build-images -r PATH_TO_THE_CHECK_DIR` it will create a docker image and
run it, feeding the check with the input from local.toml.

Full example to build and test run one check:
```
cd cmd/vulcan-drupal
go build
cp local.toml.example local.toml
./vulcan-drupal -t
```

If you are running go version 1.17 or later, you can easily use the build system project with `go run`:
```
cd cmd/vulcan-drupal
go run github.com/adevinta/vulcan-checks-bsys/cmd/vulcan-build-images@master -r ../vulcan-drupal
```

For older versions of go you should use `go install` or `go get`, depending on version.
