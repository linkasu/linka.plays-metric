locals {
  collector_domain = "plays-metric.nkolinka.ru"
  writer_domain    = "plays-metric-writer.nkolinka.ru"
  db_domain        = "plays-metric-db.nkolinka.ru"
  vps_ipv4         = "37.230.192.57"
}

resource "yandex_container_registry" "metric" {
  name = "linka-plays-metric"
}

resource "yandex_iam_service_account" "runtime" {
  name = "linka-plays-metric-runtime"
}

resource "yandex_iam_service_account" "ci" {
  name = "linka-plays-metric-ci"
}

resource "yandex_resourcemanager_folder_iam_member" "runtime_lockbox" {
  folder_id = var.folder_id
  role      = "lockbox.payloadViewer"
  member    = "serviceAccount:${yandex_iam_service_account.runtime.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "runtime_invoker" {
  folder_id = var.folder_id
  role      = "serverless-containers.containerInvoker"
  member    = "serviceAccount:${yandex_iam_service_account.runtime.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "runtime_registry" {
  folder_id = var.folder_id
  role      = "container-registry.images.puller"
  member    = "serviceAccount:${yandex_iam_service_account.runtime.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "ci_registry" {
  folder_id = var.folder_id
  role      = "container-registry.images.pusher"
  member    = "serviceAccount:${yandex_iam_service_account.ci.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "ci_containers" {
  folder_id = var.folder_id
  role      = "serverless-containers.editor"
  member    = "serviceAccount:${yandex_iam_service_account.ci.id}"
}

resource "yandex_lockbox_secret" "runtime" {
  name                = "linka-plays-metric-runtime"
  deletion_protection = true
  description         = "Payload is created and rotated outside Terraform"
}

resource "yandex_serverless_container" "collector" {
  name               = "linka-plays-metric-collector"
  memory             = 256
  cores              = 1
  core_fraction      = 100
  execution_timeout  = "15s"
  concurrency        = 16
  service_account_id = yandex_iam_service_account.runtime.id

  runtime {
    type = "http"
  }

  image {
    url = var.collector_image_url
    environment = {
      LISTEN_ADDR = ":8080"
      WRITER_URL  = var.writer_url
    }
  }

  secrets {
    id                   = yandex_lockbox_secret.runtime.id
    version_id           = var.lockbox_secret_version_id
    key                  = "installation_hmac_secret"
    environment_variable = "INSTALLATION_HMAC_SECRET"
  }

  secrets {
    id                   = yandex_lockbox_secret.runtime.id
    version_id           = var.lockbox_secret_version_id
    key                  = "writer_hmac_secret"
    environment_variable = "WRITER_HMAC_SECRET"
  }

  depends_on = [
    yandex_resourcemanager_folder_iam_member.runtime_lockbox,
    yandex_resourcemanager_folder_iam_member.runtime_registry,
  ]
}

resource "yandex_cm_certificate" "collector" {
  name    = "linka-plays-metric"
  domains = [local.collector_domain]

  managed {
    challenge_type  = "DNS_CNAME"
    challenge_count = 1
  }
}

resource "yandex_dns_recordset" "certificate_validation" {
  count   = yandex_cm_certificate.collector.managed[0].challenge_count
  zone_id = var.dns_zone_id
  name    = yandex_cm_certificate.collector.challenges[count.index].dns_name
  type    = yandex_cm_certificate.collector.challenges[count.index].dns_type
  ttl     = 60
  data    = [yandex_cm_certificate.collector.challenges[count.index].dns_value]
}

resource "yandex_api_gateway" "metric" {
  name = "linka-plays-metric"

  custom_domains {
    fqdn           = local.collector_domain
    certificate_id = yandex_cm_certificate.collector.id
  }

  spec = <<-YAML
    openapi: 3.0.0
    info:
      title: LINKa Plays Metric
      version: 1.0.0
    paths:
      /:
        x-yc-apigateway-any-method:
          x-yc-apigateway-integration:
            type: serverless_containers
            container_id: ${yandex_serverless_container.collector.id}
            service_account_id: ${yandex_iam_service_account.runtime.id}
      /{proxy+}:
        x-yc-apigateway-any-method:
          parameters:
            - name: proxy
              in: path
              required: false
              explode: false
              style: simple
              schema:
                type: string
                default: '-'
          x-yc-apigateway-integration:
            type: serverless_containers
            container_id: ${yandex_serverless_container.collector.id}
            service_account_id: ${yandex_iam_service_account.runtime.id}
  YAML

  depends_on = [
    yandex_dns_recordset.certificate_validation,
    yandex_resourcemanager_folder_iam_member.runtime_invoker,
  ]
}

resource "yandex_dns_recordset" "collector" {
  zone_id = var.dns_zone_id
  name    = "${local.collector_domain}."
  type    = "CNAME"
  ttl     = 300
  data    = ["${trimsuffix(yandex_api_gateway.metric.domain, ".")}."]
}

resource "yandex_dns_recordset" "writer" {
  zone_id = var.dns_zone_id
  name    = "${local.writer_domain}."
  type    = "A"
  ttl     = 300
  data    = [local.vps_ipv4]
}

resource "yandex_dns_recordset" "database" {
  zone_id = var.dns_zone_id
  name    = "${local.db_domain}."
  type    = "A"
  ttl     = 300
  data    = [local.vps_ipv4]
}
