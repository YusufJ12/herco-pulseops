output "container_id" {
  description = "ID kontainer PulseOps"
  value       = docker_container.pulseops.id
}

output "application_url" {
  description = "URL akses lokal dashboard PulseOps"
  value       = "http://localhost:${var.port}"
}

output "metrics_url" {
  description = "URL endpoint metrik Prometheus"
  value       = "http://localhost:${var.port}/metrics"
}
