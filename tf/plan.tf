provider "panos" {
    hostname = "${var.Hostname}"
    username = "${var.Username}"
    password = "${var.Password}"
}

resource "panos_zone" "zout" {
    name = "L3-untrust"
    mode = "layer3"
}

resource "panos_zone" "zin" {
    name = "L3-trust"
    mode = "layer3"
}

resource "panos_service_object" "app" {
    name = "cicd app"
    description = "Corporate App"
    protocol = "tcp"
    destination_port = "${var.Port}"
}

resource "panos_security_policies" "sec_rules" {
    rule {
        name = "Allow Company App"
        source_zones = ["${panos_zone.zout.name}"]
        source_addresses = ["any"]
        source_users = ["any"]
        hip_profiles = ["any"]
        destination_zones = ["${panos_zone.zin.name}"]
        destination_addresses = ["any"]
        applications = ["any"]
        services = ["${panos_service_object.app.name}"]
        categories = ["any"]
        action = "allow"
    }
    rule {
        name = "Deny everything else"
        source_zones = ["any"]
        source_addresses = ["any"]
        source_users = ["any"]
        hip_profiles = ["any"]
        destination_zones = ["any"]
        destination_addresses = ["any"]
        applications = ["any"]
        services = ["application-default"]
        categories = ["any"]
        action = "deny"
    }
}

