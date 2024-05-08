# Configure F5XC client to authenticate to API using x509 certificate and key PEM files extracted from p12.
provider "f5xc" {
  api_cert = "/path/to/cert.pem"
  api_Key  = "/path/to/key.pem"
  url      = "https://tenant.console.ves.volterra.io/api"
}
