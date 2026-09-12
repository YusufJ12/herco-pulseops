variable "app_name" {
  description = "Nama aplikasi"
  type        = string
  default     = "pulseops"
}

variable "image_tag" {
  description = "Tag versi image container"
  type        = string
  default     = "1.0.0"
}

variable "port" {
  description = "Port HTTP aplikasi"
  type        = number
  default     = 8080
}

variable "target_url" {
  description = "Default URL target yang dipantau"
  type        = string
  default     = "https://portfolioyusufjaelani.vercel.app"
}

variable "target_name" {
  description = "Default label target yang dipantau"
  type        = string
  default     = "Yusuf Portfolio"
}

variable "probe_interval_seconds" {
  description = "Interval probe dalam detik"
  type        = number
  default     = 30
}
