provider "aws" {
    access_key = "${var.aws_access_key}"
    secret_key = "${var.aws_secret_key}"
    region = "${var.aws_region}"
}

resource "random_string" "randPrefix" {
    length = 1
    lower = true
    upper = false
    number = false
    special = false
}

resource "random_string" "randSuffix" {
    length = 9
    override_special = "!@#%^&*()[]/?|,.-_+:;"
}

resource "random_string" "sgName" {
    length = 10
    number = false
    special = false
}

resource "aws_security_group" "sg" {
    name = "${random_string.sgName.result}"
    description = "cloud automation demo sg"

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
        description = "Allow SSH"
    }

    ingress {
        from_port = 443
        to_port = 443
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
        description = "Allow HTTPS"
    }

    ingress {
        from_port = 8080
        to_port = 8080
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
        description = "Allow webhook listener"
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_instance" "panos" {
    ami = "${var.panos_ami}"
    instance_type = "m4.xlarge"
    key_name = "${var.aws_ssh_key_name}"
    security_groups = ["${aws_security_group.sg.name}"]

    ebs_block_device {
        device_name = "/dev/xvda"
        volume_type = "gp2"
        delete_on_termination = true
        volume_size = 60
    }
}

resource "aws_instance" "linux" {
    ami = "${var.linux_ami}"
    instance_type = "${var.linux_instance_type}"
    key_name = "${var.aws_ssh_key_name}"
    security_groups = ["${aws_security_group.sg.name}"]
    user_data = <<INIT
#!/bin/bash
echo "Starting user data config initialization"
cd /home/ec2-user
echo "Saving panos info"
echo '{' > config.json
echo '  "github_account": "${var.github_account}",' >> config.json
echo '  "hostname": "${aws_instance.panos.public_ip}",' >> config.json
echo '  "username": "${var.panos_username}",' >> config.json
echo '  "password": "${random_string.randPrefix.result}${random_string.randSuffix.result}"' >> config.json
echo '}' >> config.json
echo "Making required directories ..."
mkdir bin
mkdir tf
mkdir an
mkdir golang
mkdir golang/bin
mkdir golang/src
echo "Updating .bash_profile ..."
echo 'export GOPATH=/home/ec2-user/golang' >> /home/ec2-user/.bash_profile
echo 'export GOBIN=/home/ec2-user/golang/bin' >> /home/ec2-user/.bash_profile
echo "alias s='cd ..'" >> /home/ec2-user/.bash_profile
echo "alias la='ls -laF'" >> /home/ec2-user/.bash_profile
echo "alias wl='tail -f /tmp/hook.log'" >> /home/ec2-user/.bash_profile
echo "Updating yum and installing golang ..."
yum update -y
yum install -y golang
echo "Pulling down the github repo to watch ..."
git clone https://github.com/PaloAltoNetworks/cloud-automation-demo.git
cp -r cloud-automation-demo/tf .
cp -r cloud-automation-demo/an .
echo "Ansible: install and prep ..."
pip install pan-python pandevice ansible
echo "Terraform: install and prep ..."
cd bin
curl -o tf.zip https://releases.hashicorp.com/terraform/0.11.4/terraform_0.11.4_linux_amd64.zip
unzip tf.zip
rm -f tf.zip
cd ..
echo "Building webhook listener ..."
touch /tmp/hook.log
cd cloud-automation-demo
go build -o ../bin/las cloud-automation-demo/las.go
cd ..
echo "Fixing all permissions ..."
chown -R ec2-user:ec2-user /home/ec2-user
chown ec2-user:ec2-user /tmp/hook.log
chmod 666 /tmp/hook.log
echo "Launching webhook listener ..."
/home/ec2-user/bin/las &
echo "Done with user data init!"
INIT
}

provider "github" {
    token = "${var.github_token}"
    organization = "PaloAltoNetworks"
}

resource "github_repository_webhook" "hook" {
    repository = "cloud-automation-demo"
    name = "web"
    events = ["push"]
    configuration {
        url = "http://${aws_instance.linux.public_ip}:8080/"
        content_type = "json"
    }
}

resource "null_resource" "fwinit" {
    triggers {
        key = "${aws_instance.panos.public_ip}"
    }

    provisioner "local-exec" {
        command = "./fw_init.sh ${aws_instance.panos.public_ip} ${var.panos_username} '${random_string.randPrefix.result}${random_string.randSuffix.result}'"
    }
}


output "panos_ip" {
    value = "${aws_instance.panos.public_ip}"
}

output "panos_password" {
    value = "${random_string.randPrefix.result}${random_string.randSuffix.result}"
}

output "linux_ip" {
    value = "${aws_instance.linux.public_ip}"
}
