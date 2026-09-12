terraform {
  required_version = ">= 1.5.0"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0.2"
    }
  }
}

provider "docker" {}

resource "docker_volume" "pulseops_data" {
  name = "${var.app_name}_data"
}

resource "docker_image" "pulseops" {
  name         = "ghcr.io/yusufj12/pulseops:${var.image_tag}"
  keep_locally = true
}

resource "docker_container" "pulseops" {
  name  = var.app_name
  image = docker_image.pulseops.image_id

  restart = "unless-stopped"

  ports {
    internal = var.port
    external = var.port
  }

  env = [
    "PORT=${var.port}",
    "DB_PATH=/data/pulseops.db",
    "PROBE_INTERVAL_SECONDS=${var.probe_interval_seconds}",
    "DEFAULT_TARGET_URL=${var.target_url}",
    "DEFAULT_TARGET_NAME=${var.target_name}"
  ]

  volumes {
    volume_name    = docker_volume.pulseops_data.name
    container_path = "/data"
  }

  healthcheck {
    test         = ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:${var.port}/healthz"]
    interval     = "15s"
    timeout      = "5s"
    retries      = 3
    start_period = "5s"
  }
}
