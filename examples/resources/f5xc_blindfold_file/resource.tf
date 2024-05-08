# Blindfold an existing JSON Google service account key and register as a Cloud Credential to provision GCP VPC sites.

resource "f5xc_blindfold_file" "creds" {
  path = "/path/to/sa.json"
  policy_document = {
    name      = "ves-io-allow-volterra"
    namespace = "shared"
  }
}

resource "volterra_cloud_credential" "gcp" {
  name        = "gcp"
  namespace   = "system"
  description = "GCP SA"
  gcp_cred_file {
    credential_file {
      blindfold_secret_info {
        location = format("string:///%s", f5xc_blindfold_file.creds.sealed)
      }
    }
  }
}
