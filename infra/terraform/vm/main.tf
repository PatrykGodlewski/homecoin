data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "lab" {
  count    = var.create_resource_group ? 1 : 0
  name     = var.resource_group_name
  location = var.location
  tags     = var.tags
}

locals {
  resource_group_name = var.create_resource_group ? azurerm_resource_group.lab[0].name : var.resource_group_name
  resource_group_id   = var.create_resource_group ? azurerm_resource_group.lab[0].id : "/subscriptions/${data.azurerm_client_config.current.subscription_id}/resourceGroups/${var.resource_group_name}"
  location            = var.create_resource_group ? azurerm_resource_group.lab[0].location : var.location
  vnet_cidr           = "172.21.0.0/16"
  subnet_cidr         = "172.21.0.0/24"
}

resource "tls_private_key" "vm" {
  count     = var.ssh_public_key == "" ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

locals {
  ssh_public_key = var.ssh_public_key != "" ? var.ssh_public_key : tls_private_key.vm[0].public_key_openssh
}

resource "azurerm_virtual_network" "lab" {
  name                = "${var.project_name}-vnet"
  address_space       = [local.vnet_cidr]
  location            = local.location
  resource_group_name = local.resource_group_name
  tags                = var.tags
}

resource "azurerm_subnet" "web" {
  name                 = "serwery_www"
  resource_group_name  = local.resource_group_name
  virtual_network_name = azurerm_virtual_network.lab.name
  address_prefixes     = [local.subnet_cidr]
}

resource "azurerm_network_security_group" "lab" {
  name                = "${var.project_name}-nsg"
  location            = local.location
  resource_group_name = local.resource_group_name
  tags                = var.tags

  security_rule {
    name                       = "SSH"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = var.allowed_ssh_cidr
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "HTTP"
    priority                   = 101
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "HTTPS"
    priority                   = 102
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "web" {
  subnet_id                 = azurerm_subnet.web.id
  network_security_group_id = azurerm_network_security_group.lab.id
}

resource "azurerm_public_ip" "vm" {
  count               = var.vm_count
  name                = "${var.project_name}-pip-${count.index}"
  location            = local.location
  resource_group_name = local.resource_group_name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = var.tags
}

resource "azurerm_network_interface" "vm" {
  count               = var.vm_count
  name                = "${var.project_name}-nic-${count.index}"
  location            = local.location
  resource_group_name = local.resource_group_name
  tags                = var.tags

  ip_configuration {
    name                          = "primary"
    subnet_id                     = azurerm_subnet.web.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.vm[count.index].id
  }
}

resource "azurerm_linux_virtual_machine" "web" {
  count               = var.vm_count
  name                = "${var.project_name}-app-${count.index}"
  location            = local.location
  resource_group_name = local.resource_group_name
  size                = var.vm_size
  tags                = var.tags

  admin_username                  = var.admin_username
  disable_password_authentication = true

  network_interface_ids = [azurerm_network_interface.vm[count.index].id]

  admin_ssh_key {
    username   = var.admin_username
    public_key = local.ssh_public_key
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }
}
