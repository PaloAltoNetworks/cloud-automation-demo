resource "panos_ethernet_interface" "eth1" {
    name = "ethernet1/1"
    mode = "layer3"
    vsys = "vsys1"
}

resource "panos_zone" "zut" {
    name = "L3-untrust"
    mode = "layer3"
    interfaces = [panos_ethernet_interface.eth1.name]
}

resource "panos_ethernet_interface" "eth2" {
    name = "ethernet1/2"
    mode = "layer3"
    vsys = "vsys1"
}

resource "panos_zone" "zt" {
    name = "L3-trust"
    mode = "layer3"
    interfaces = [panos_ethernet_interface.eth2.name]
}
