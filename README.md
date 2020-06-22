# GoCmdScanner
Golang script to perform code review across projects by wrapping around `grep` binary.

The project is inspired by the [nuclei](https://github.com/projectdiscovery/nuclei) project.

## Examples

### Standard Usage
Assuming a signature file as follows, grep is used to recursively serach for PHP SQL Injection vulnerabilities using regex format listed in the signature file.

Results are written to `out-php-sqlinjection.txt` inside folder `out-codereview` by default

```
$ cat php_sqlinjection.yaml
id: "php_sqlinjection"
name: "PHP SQL Injection"
author: "manasmbellani"
severity: "high"
checks: 
    - outfile: out-php-sqlinjection.txt
      regex:
        - "sqli"
        - "pgsql"
      notes: >
        Commmon PHP SQL Injection sink functions as searched above can indicate 
        vulnerabilities. 


$ go run gocodereview.go -f /tmp/test.txt -s /opt/athena-tools-private/codereview/cmdscanner_templates/php_sqlinjection.yaml
$ ls -1 out-codereview/
out-php-sqlinjection.txt
```
