# Terraform bootstrap YC

Конфигурация создаёт Container Registry, service accounts runtime/CI, пустой Lockbox secret, Serverless Container, API Gateway, управляемый сертификат и DNS-записи. Payload Lockbox намеренно не описан Terraform и не попадает в state.

## Порядок bootstrap

1. Создайте закрытый versioned bucket `linka-plays-metric-tfstate-b1gn4stour811vgtjude`, выдайте CI service account роль `storage.editor` и передайте его static access key через `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`.
2. Передайте `cloud_id`, `folder_id`, `zone`, `dns_zone_id`, immutable `collector_image_url`.
3. Создайте только контейнер Lockbox: `terraform apply -target=yandex_lockbox_secret.runtime`.
4. В UI или YC CLI создайте версию secret с ключами `installation_hmac_secret` и `writer_hmac_secret`, оба минимум 32 случайных байта.
5. Передайте только ID версии как `lockbox_secret_version_id` и выполните обычные `terraform plan`/`terraform apply` после проверки плана.
6. Отдельно примените certificate challenge record: `terraform apply -target=yandex_dns_recordset.certificate_validation`, дождитесь статуса сертификата `ISSUED`, затем выполните полный apply. Это исключает гонку между выдачей сертификата и созданием custom domain API Gateway.

Репозиторий не выполняет `terraform apply`. Не передавайте payload через `.tfvars`: даже sensitive variables сохраняются в state. `terraform.tfvars.example` содержит только фиктивные идентификаторы.

Serverless Containers обычно получает образы из Yandex Container Registry. CI публикует GHCR-образ; перед deploy его следует зеркалировать в созданный registry отдельным доверенным CI job без сборки либо указать уже доступный `cr.yandex/...` URL.

CI service account получает минимальные folder-роли для push образов и обновления container. Создание ключа service account и его помещение в GitHub secrets выполняются вне Terraform, чтобы закрытый ключ не оказался в state.
