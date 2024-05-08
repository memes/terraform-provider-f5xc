# Configure F5XC client to authenticate to API using PKCS#12 file downloaded from console.
# NOTE: The VES_P12_PASSWORD environment variable must contain the passphrase associated with the P12 file.
provider "f5xc" {
  api_p12_file = "/path/to/auth.p12"
  url          = "https://tenant.console.ves.volterra.io/api"
}
