package main

import (
	"fmt"
	"os"

	"github.com/jxskiss/mcli"
)

/*
NAME:
   lego - Let's Encrypt client written in Go

USAGE:
   lego [global options] command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
   run      Register an account, then create and install a certificate
   revoke   Revoke a certificate
   renew    Renew a certificate
   dnshelp  Shows additional help for the '--dns' global option
   list     Display certificates and accounts information.
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --domains value, -d value    Add a domain to the process. Can be specified multiple times.
   --server value, -s value     CA hostname (and optionally :port). The server certificate must be trusted in order to avoid further modifications to the client. (default: "https://acme-v02.api.letsencrypt.org/directory")
   --accept-tos, -a             By setting this flag to true you indicate that you accept the current Let's Encrypt terms of service. (default: false)
   --email value, -m value      Email used for registration and recovery contact.
   --csr value, -c value        Certificate signing request filename, if an external CSR is to be used.
   --eab                        Use External Account Binding for account registration. Requires --kid and --hmac. (default: false)
   --kid value                  Key identifier from External CA. Used for External Account Binding.
   --hmac value                 MAC key from External CA. Should be in Base64 URL Encoding without padding format. Used for External Account Binding.
   --key-type value, -k value   Key type to use for private keys. Supported: rsa2048, rsa4096, rsa8192, ec256, ec384. (default: "ec256")
   --filename value             (deprecated) Filename of the generated certificate.
   --path value                 Directory to use for storing the data. (default: "/Users/wsh/go/src/github.com/jxskiss/mcli/.lego") [$LEGO_PATH]
   --http                       Use the HTTP challenge to solve challenges. Can be mixed with other types of challenges. (default: false)
   --http.port value            Set the port and interface to use for HTTP based challenges to listen on.Supported: interface:port or :port. (default: ":80")
   --http.proxy-header value    Validate against this HTTP header when solving HTTP based challenges behind a reverse proxy. (default: "Host")
   --http.webroot value         Set the webroot folder to use for HTTP based challenges to write directly in a file in .well-known/acme-challenge. This disables the built-in server and expects the given directory to be publicly served with access to .well-known/acme-challenge
   --http.memcached-host value  Set the memcached host(s) to use for HTTP based challenges. Challenges will be written to all specified hosts.
   --tls                        Use the TLS challenge to solve challenges. Can be mixed with other types of challenges. (default: false)
   --tls.port value             Set the port and interface to use for TLS based challenges to listen on. Supported: interface:port or :port. (default: ":443")
   --dns value                  Solve a DNS challenge using the specified provider. Can be mixed with other types of challenges. Run 'lego dnshelp' for help on usage.
   --dns.disable-cp             By setting this flag to true, disables the need to wait the propagation of the TXT record to all authoritative name servers. (default: false)
   --dns.resolvers value        Set the resolvers to use for performing recursive DNS queries. Supported: host:port. The default is to use the system resolvers, or Google's DNS resolvers if the system's cannot be determined.
   --http-timeout value         Set the HTTP timeout value to a specific value in seconds. (default: 0)
   --dns-timeout value          Set the DNS timeout value to a specific value in seconds. Used only when performing authoritative name servers queries. (default: 10)
   --pem                        Generate a .pem file by concatenating the .key and .crt files together. (default: false)
   --pfx                        Generate a .pfx (PKCS#12) file by with the .key and .crt and issuer .crt files together. (default: false)
   --pfx.pass value             The password used to encrypt the .pfx (PCKS#12) file. (default: "changeit")
   --cert.timeout value         Set the certificate timeout value to a specific value in seconds. Only used when obtaining certificates. (default: 30)
   --help, -h                   show help (default: false)
   --version, -v                print the version (default: false)

*/

var globalFlags GlobalFlags

func main() {
	app := &mcli.App{
		Description: `hugo is the main command, used to build your Hugo site.

Hugo is a Fast and Flexible Static Site Generator
built with love by spf13 and friends in Go.

Complete documentation is available at http://gohugo.io/.`,
	}
	app.SetGlobalFlags(&globalFlags)
	app.AddRoot(cmdRoot)
	app.Add("run", cmdRun, "Register an account, then create and install a certificate")
	app.Add("revoke", cmdRevoke, "Revoke a certificate")
	app.Add("renew", cmdRenew, "Renew a certificate")
	app.Add("dnshelp", cmdDnshelp, "Shows additional help for the '--dns' global option")
	app.Add("list", cmdList, "Display certificates and accounts information")
	app.AddCompletion()
	app.Options.EnableFlagCompletionForAllCommands = true
	app.Run()
}

type GlobalFlags struct {
	Domains           []string `cli:"-d, --domain, Add a domain to the process. Can be specified multiple times."`
	Server            string   `cli:"-s, --server, CA hostname (and optionally :port). The server certificate must be trusted in order to avoid further modifications to the client." default:"https://acme-v02.api.letsencrypt.org/directory"`
	AcceptTOS         bool     `cli:"-a, --accept-tos, By setting this flag to true you indicate that you accept the current Let's Encrypt terms of service."`
	Email             string   `cli:"-m, --email, Email used for registration and recovery contact."`
	CSR               string   `cli:"-c, --csr, Certificate signing request filename, if an external CSR is to be used."`
	EAB               bool     `cli:"--eab, Use External Account Binding for account registration. Requires --kid and --hmac."`
	KID               string   `cli:"--kid, Key identifier from External CA. Used for External Account Binding."`
	HMAC              string   `cli:"--hamc, MAC key from External CA. Should be in Base64 URL Encoding without padding format. Used for External Account Binding."`
	KeyType           string   `cli:"-k, --key-type, Key type to use for private keys. Supported: rsa2048, rsa4096, rsa8192, ec256, ec384." default:"ec256"`
	Filename          string   `cli:"#D, --filename, Filename of the generated certificate."`
	Path              string   `cli:"--path, Directory to use for storing the data." env:"LEGO_PATH"`
	HTTP              bool     `cli:"--http, Use the HTTP challenge to solve challenges. Can be mixed with other types of challenges."`
	HttpPort          string   `clI:"--http.port, Set the port and interface to use for HTTP based challenges to listen on.Supported: interface:port or :port." default:":80"`
	HttpProxyHeader   string   `cli:"--http.proxy-header, Validate against this HTTP header when solving HTTP based challenges behind a reverse proxy." default:"Host"`
	HttpWebroot       string   `cli:"--http.webroot, Set the webroot folder to use for HTTP based challenges to write directly in a file in .well-known/acme-challenge. This disables the built-in server and expects the given directory to be publicly served with access to .well-known/acme-challenge"`
	HttpMemcachedHost []string `cli:"--http.memcached-host, Set the memcached host(s) to use for HTTP based challenges. Challenges will be written to all specified hosts."`
	TLS               bool     `cli:"--tls, Use the TLS challenge to solve challenges. Can be mixed with other types of challenges."`
	TlsPort           string   `cli:"--tls.port, Set the port and interface to use for TLS based challenges to listen on. Supported: interface:port or :port." default:":443"`
	DNS               string   `cli:"--dns, Solve a DNS challenge using the specified provider. Can be mixed with other types of challenges. Run 'lego dnshelp' for help on usage."`
	DnsDisableCp      bool     `cli:"--dns.disable-cp, By setting this flag to true, disables the need to wait the propagation of the TXT record to all authoritative name servers."`
	DnsResolvers      []string `cli:"--dns.resolvers, Set the resolvers to use for performing recursive DNS queries. Supported: host:port. The default is to use the system resolvers, or Google\\'s DNS resolvers if the system\\'s cannot be determined."`
	HttpTimeout       int      `cli:"--http-timeout, Set the HTTP timeout value to a specific value in seconds."`
	DnsTimeout        int      `cli:"--dns-timeout, Set the DNS timeout value to a specific value in seconds. Used only when performing authoritative name servers queries."`
	PEM               bool     `cli:"--pem, Generate a .pem file by concatenating the .key and .crt files together."`
	PFX               bool     `cli:"--pfx, Generate a .pfx (PKCS#12) file by with the .key and .crt and issuer .crt files together."`
	PfxPass           string   `cli:"--pfx.pass, The password used to encrypt the .pfx (PCKS#12) file." default:"changeit"`
	CertTimeout       int      `cli:"--cert.timeout, Set the certificate timeout value to a specific value in seconds. Only used when obtaining certificates." default:"30"`
}

type CommonCertFlags struct {
	NoBundle       bool   `cli:"--no-bundle, Do not create a certificate bundle by adding the issuers certificate to the new certificate."`
	MustStaple     bool   `cli:"--must-staple, Include the OCSP must staple TLS extension in the CSR and generated certificate. Only works if the CSR is generated by lego."`
	PreferredChain string `cli:"--preferred-chain, If the CA offers multiple certificate chains, prefer the chain with an issuer matching this Subject Common Name. If no match, the default offered chain will be used."`

	AlwaysDeactivateAuthorization string `cli:"--always-deactivate-authorizations, Force the authorizations to be relinquished even if the certificate request was successful."`
}

/*
NAME:
   lego run - Register an account, then create and install a certificate

USAGE:
   lego run [command options] [arguments...]

OPTIONS:
   --no-bundle                               Do not create a certificate bundle by adding the issuers certificate to the new certificate. (default: false)
   --must-staple                             Include the OCSP must staple TLS extension in the CSR and generated certificate. Only works if the CSR is generated by lego. (default: false)
   --run-hook value                          Define a hook. The hook is executed when the certificates are effectively created.
   --preferred-chain value                   If the CA offers multiple certificate chains, prefer the chain with an issuer matching this Subject Common Name. If no match, the default offered chain will be used.
   --always-deactivate-authorizations value  Force the authorizations to be relinquished even if the certificate request was successful.
   --help, -h                                show help (default: false)

*/

func cmdRun(ctx *mcli.Context) {
	var args struct {
		CommonCertFlags
		Hook string `cli:"--run-hook, Define a hook. The hook is executed when the certificates are effectively created."`
	}
	ctx.Parse(&args)
	ctx.PrintHelp()
}

/*
NAME:
   lego revoke - Revoke a certificate

USAGE:
   lego revoke [command options] [arguments...]

OPTIONS:
   --keep, -k      Keep the certificates after the revocation instead of archiving them. (default: false)
   --reason value  Identifies the reason for the certificate revocation. See https://www.rfc-editor.org/rfc/rfc5280.html#section-5.3.1. 0(unspecified),1(keyCompromise),2(cACompromise),3(affiliationChanged),4(superseded),5(cessationOfOperation),6(certificateHold),8(removeFromCRL),9(privilegeWithdrawn),10(aACompromise) (default: 0)
   --help, -h      show help (default: false)

*/

func cmdRevoke(ctx *mcli.Context) {
	var args struct {
		Keep   bool `cli:"--keep, Keep the certificates after the revocation instead of archiving them."`
		Reason int  `cli:"--reason, Identifies the reason for the certificate revocation. See https://www.rfc-editor.org/rfc/rfc5280.html#section-5.3.1. 0(unspecified),1(keyCompromise),2(cACompromise),3(affiliationChanged),4(superseded),5(cessationOfOperation),6(certificateHold),8(removeFromCRL),9(privilegeWithdrawn),10(aACompromise)"`
	}
	ctx.Parse(&args)
	ctx.PrintHelp()
}

/*
NAME:
   lego renew - Renew a certificate

USAGE:
   lego renew [command options] [arguments...]

OPTIONS:
   --days value                              The number of days left on a certificate to renew it. (default: 30)
   --reuse-key                               Used to indicate you want to reuse your current private key for the new certificate. (default: false)
   --no-bundle                               Do not create a certificate bundle by adding the issuers certificate to the new certificate. (default: false)
   --must-staple                             Include the OCSP must staple TLS extension in the CSR and generated certificate. Only works if the CSR is generated by lego. (default: false)
   --renew-hook value                        Define a hook. The hook is executed only when the certificates are effectively renewed.
   --preferred-chain value                   If the CA offers multiple certificate chains, prefer the chain with an issuer matching this Subject Common Name. If no match, the default offered chain will be used.
   --always-deactivate-authorizations value  Force the authorizations to be relinquished even if the certificate request was successful.
   --help, -h                                show help (default: false)

*/

func cmdRenew(ctx *mcli.Context) {
	var args struct {
		CommonCertFlags
		Days      int    `cli:"--days, The number of days left on a certificate to renew it." default:"30"`
		ReuseKey  bool   `cli:"--reuse-key, Used to indicate you want to reuse your current private key for the new certificate."`
		RenewHook string `cli:"--renew-hook, Define a hook. The hook is executed only when the certificates are effectively renewed."`
	}
	ctx.Parse(&args)
	ctx.PrintHelp()
}

/*
Credentials for DNS providers must be passed through environment variables.

To display the documentation for a DNS providers:

  $ lego dnshelp -c code

All DNS codes:
  acme-dns, alidns, allinkl, arvancloud, auroradns, autodns, azure, bindman, bluecat, checkdomain, clouddns, cloudflare, cloudns, cloudxns, conoha, constellix, desec, designate, digitalocean, dnsimple, dnsmadeeasy, dnspod, dode, domeneshop, dreamhost, duckdns, dyn, dynu, easydns, edgedns, epik, exec, exoscale, freemyip, gandi, gandiv5, gcloud, gcore, glesys, godaddy, hetzner, hostingde, hosttech, httpreq, hurricane, hyperone, ibmcloud, iij, iijdpf, infoblox, infomaniak, inwx, ionos, iwantmyname, joker, lightsail, linode, liquidweb, loopia, luadns, manual, mydnsjp, mythicbeasts, namecheap, namedotcom, namesilo, netcup, netlify, nicmanager, nifcloud, njalla, ns1, oraclecloud, otc, ovh, pdns, porkbun, rackspace, regru, rfc2136, rimuhosting, route53, safedns, sakuracloud, scaleway, selectel, servercow, simply, sonic, stackpath, tencentcloud, transip, vegadns, vercel, versio, vinyldns, vscale, vultr, wedos, yandex, zoneee, zonomi

More information: https://go-acme.github.io/lego/dns

*/

func cmdDnshelp(ctx *mcli.Context) {
	var args struct {
		Code string `cli:"-c"`
	}
	ctx.Parse(&args, mcli.DisableGlobalFlags(), mcli.ReplaceUsage(func() string {
		return `
Credentials for DNS providers must be passed through environment variables.

To display the documentation for a DNS providers:

  $ lego dnshelp -c code

All DNS codes:
  acme-dns, alidns, allinkl, arvancloud, auroradns, autodns, azure, bindman, bluecat, checkdomain, clouddns, cloudflare, cloudns, cloudxns, conoha, constellix, desec, designate, digitalocean, dnsimple, dnsmadeeasy, dnspod, dode, domeneshop, dreamhost, duckdns, dyn, dynu, easydns, edgedns, epik, exec, exoscale, freemyip, gandi, gandiv5, gcloud, gcore, glesys, godaddy, hetzner, hostingde, hosttech, httpreq, hurricane, hyperone, ibmcloud, iij, iijdpf, infoblox, infomaniak, inwx, ionos, iwantmyname, joker, lightsail, linode, liquidweb, loopia, luadns, manual, mydnsjp, mythicbeasts, namecheap, namedotcom, namesilo, netcup, netlify, nicmanager, nifcloud, njalla, ns1, oraclecloud, otc, ovh, pdns, porkbun, rackspace, regru, rfc2136, rimuhosting, route53, safedns, sakuracloud, scaleway, selectel, servercow, simply, sonic, stackpath, tencentcloud, transip, vegadns, vercel, versio, vinyldns, vscale, vultr, wedos, yandex, zoneee, zonomi

More information: https://go-acme.github.io/lego/dns
`
	}))

	if args.Code == "" {
		ctx.PrintHelp()
	}

	fmt.Println(os.Args)
}

/*
NAME:
   lego list - Display certificates and accounts information.

USAGE:
   lego list [command options] [arguments...]

OPTIONS:
   --accounts, -a  Display accounts. (default: false)
   --names, -n     Display certificate common names only. (default: false)
   --help, -h      show help (default: false)

*/

func cmdList(ctx *mcli.Context) {
	var args struct {
		Accounts bool `cli:"-a, --accounts, Display accounts."`
		Names    bool `cli:"-n, --names, Display certificate common names only."`
	}
	ctx.Parse(&args, mcli.DisableGlobalFlags())
	ctx.PrintHelp()
}

func cmdRoot(ctx *mcli.Context) {
	var args struct {
		Names bool   `cli:"-n, --names, Display certificate common names only."`
		Arg   string `cli:"arg, Dummy demo arg"`
	}
	ctx.Parse(&args, mcli.DisableGlobalFlags())
	ctx.PrintHelp()
}
