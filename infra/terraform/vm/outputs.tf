output "resource_group_name" {
  value = local.resource_group_name
}

output "public_ips" {
  description = "Public IP addresses of application VMs"
  value       = azurerm_public_ip.vm[*].ip_address
}

output "private_ips" {
  description = "Private IP addresses of application VMs"
  value       = azurerm_network_interface.vm[*].private_ip_address
}

output "vm_names" {
  value = azurerm_linux_virtual_machine.web[*].name
}

output "admin_username" {
  value = var.admin_username
}

output "ssh_private_key_pem" {
  description = "Generated SSH private key (empty when ssh_public_key was provided)"
  value       = var.ssh_public_key == "" ? tls_private_key.vm[0].private_key_pem : ""
  sensitive   = true
}

output "ansible_inventory" {
  description = "Ansible inventory snippet (INI format)"
  value = join("\n", [
    for i in range(var.vm_count) :
    "${azurerm_linux_virtual_machine.web[i].name} ansible_host=${azurerm_public_ip.vm[i].ip_address} ansible_user=${var.admin_username} private_ip=${azurerm_network_interface.vm[i].private_ip_address}"
  ])
}

output "app_urls" {
  description = "HomeCoin HTTPS endpoints"
  value       = [for ip in azurerm_public_ip.vm[*].ip_address : "https://${ip}/"]
}

output "health_urls" {
  value = [for ip in azurerm_public_ip.vm[*].ip_address : "https://${ip}/health"]
}
