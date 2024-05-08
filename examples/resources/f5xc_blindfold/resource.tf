# Create a new Google service account key, blindfold it, and register as a Cloud Credential to provision GCP VPC sites.

resource "google_service_account_key" "sa" {
  service_account_id = var.sa_id
  key_algorithm      = "KEY_ALG_RSA_2048"
  private_key_type   = "TYPE_GOOGLE_CREDENTIALS_FILE"
}

resource "f5xc_blindfold" "creds" {
  plaintext = google_service_account_key.sa.private_key
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
        location = format("string:///%s", f5xc_blindfold.creds.sealed)
      }
    }
  }
}
