# Configure F5XC client to authenticate to API using token downloaded from console.
provider "f5xc" {
  api_token = "...base64 API token..."
  url       = "https://tenant.console.ves.volterra.io/api"
}
