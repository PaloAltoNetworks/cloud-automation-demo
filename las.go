package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
)

const BaseDir string = "/home/ec2-user"
const RepoName string = "cloud-automation-demo"
const RepoDir string = "/home/ec2-user/cloud-automation-demo"
const TerraformBinary string = "/home/ec2-user/bin/terraform"
const CommitBinary string = "/home/ec2-user/bin/commit"

type Ping struct {
    Hook HookInfo `json:"hook"`
    Zen string `json:"zen"`
}

type HookInfo struct {
    Id int `json:"id"`
    Name string `json:"name"`
    PingUrl string `json:"ping_url"`
}

type Payload struct {
    Repo Repository `json:"repository"`
    From Pusher `json:"pusher"`
}

func (p *Payload) IsValid() error {
    log.Printf("Validating payload ...")
    if p.Repo.Name != fmt.Sprintf("PaloAltoNetworks/%s", RepoName) {
        return fmt.Errorf("Invalid repo name")
    } else if p.From.Name != config.GitHubAccount {
        return fmt.Errorf("Skipping other user commit")
    }

    return nil
}

type Repository struct {
    Name string `json:"full_name"`
    Url string `json:"html_url"`
    Git string `json:"git_url"`
}

type Pusher struct {
    Name string `json:"name"`
}

type DemoConfig struct {
    Port int `json:"port"`
    Method string `json:"exec"`
}

type HookConfig struct {
    Hostname string `json:"hostname"`
    Username string `json:"username"`
    Password string `json:"password"`
    GitHubAccount string `json:"github_account"`
}

// Global variables.
var config HookConfig
var lf *os.File
var deployed bool


func handleReq(w http.ResponseWriter, r *http.Request) {
    var err error

    log.Printf("New [push] detected")
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error in readall: %s", err)
        return
    }

    // Check if it's a [ping] event.
    p := Ping{}
    if err = json.Unmarshal(body, &p); err == nil && p.Zen != "" {
        log.Printf("Got Ping event: id:%d name:%s url:%s", p.Hook.Id, p.Hook.Name, p.Hook.PingUrl)
        log.Printf("Zen quote: %s", p.Zen)
        return
    }

    // Unmarshal the [push] event.
    data := Payload{}
    if err = json.Unmarshal(body, &data); err != nil {
        log.Printf("Unmarshal failed (invalid request): %s", err)
        log.Printf("Raw data: %s", body)
        return
    }
    if err = data.IsValid(); err != nil {
        log.Printf("%s", err)
        return
    }

    // Chdir to the git repo and git pull.
    log.Printf("Updating local %q ...", data.Repo.Name)
    if err = os.Chdir(RepoDir); err != nil {
        log.Printf("Failed cd: %s", err)
        return
    }
    c1 := exec.Command("git", "pull")
    c1.Stdout, c1.Stderr = lf, lf
    err = c1.Run()
    if err != nil {
        log.Printf("git pull failed: %s", err)
        return
    }

    // Read the config from the repo.
    log.Printf("Reading settings.json ...")
    fd, err := os.Open("settings.json")
    if err != nil {
        log.Printf("Failed to open settings.json: %s", err)
        return
    }
    defer fd.Close()
    body, err = ioutil.ReadAll(fd)
    if err != nil {
        log.Printf("Failed to read settings.json: %s", err)
        return
    }
    demo := DemoConfig{}
    if err = json.Unmarshal(body, &demo); err != nil || demo.Port == 0 || demo.Method == "" {
        log.Printf("Failed to parse demo config: %s", err)
        return
    }

    // Perform the requested demo.
    if demo.Method == "ansible" {
        dstDir := fmt.Sprintf("%s/an", BaseDir)

        log.Printf("Copying ansible files into place ...")
        if err = copyFiles(fmt.Sprintf("%s/an", RepoDir), dstDir); err != nil {
            log.Printf("%s", err)
            return
        }

        log.Printf("Creating ansible playbooks ...")
        if err = os.Chdir(BaseDir + "/an"); err != nil {
            log.Printf("Failed cd: %s", err)
            return
        }
        fd, err = os.OpenFile("fw_creds.yml", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
        if err != nil {
            log.Printf("Failed to open fw_creds.yml: %s", err)
            return
        }
        fmt.Fprintf(fd, fmt.Sprintf(`
ip_address: '%s'
username: '%s'
password: '%s'
`, config.Hostname, config.Username, config.Password))
        fd.Close()

        log.Printf("Running Ansible to configure the firewall ...")
        log.Printf("Done!")
        if !deployed {
            deployed = true
        }
    } else if demo.Method == "terraform" {
        dstDir := fmt.Sprintf("%s/tf", BaseDir)

        log.Printf("Copying terraform files into place ...")
        if err = copyFiles(fmt.Sprintf("%s/tf", RepoDir), dstDir); err != nil {
            log.Printf("%s", err)
            return
        }

        log.Printf("Updating terraform variables ...")
        if err = os.Chdir(dstDir); err != nil {
            log.Printf("Failed cd: %s", err)
            return
        }
        fd, err = os.OpenFile("terraform.tfvars", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
        if err != nil {
            log.Printf("Failed to open terraform.tfvars: %s", err)
            return
        }
        fmt.Fprintf(fd, fmt.Sprintf(`
Hostname = %q
Username = %q
Password = %q
Port = "%d"
`, config.Hostname, config.Username, config.Password, demo.Port))
        fd.Close()

        log.Printf("Running Terraform to configure the firewall ...")
        c3 := exec.Command(TerraformBinary, "init")
        c3.Stdout, c3.Stderr = lf, lf
        if err = c3.Run(); err != nil {
            log.Printf("Failed to run terraform init: %s", err)
            return
        }
        c4 := exec.Command(TerraformBinary, "apply", "-auto-approve")
        c4.Stdout, c4.Stderr = lf, lf
        if err = c4.Run(); err != nil {
            log.Printf("Failed to run terraform apply: %s", err)
            return
        }
        c5 := exec.Command(CommitBinary)
        c5.Stdout, c5.Stderr = lf, lf
        if err = c5.Run(); err != nil {
            log.Printf("Failed to commit: %s", err)
            return
        }

        log.Printf("Done!")
        if !deployed {
            deployed = true
        }
    } else {
        log.Printf("Unknown demo method: %s", demo.Method)
    }

    fmt.Fprintf(w, "Hello, world!")
}

func copyFiles(src, dst string) error {
    files, err := ioutil.ReadDir(src)
    if err != nil {
        return fmt.Errorf("Error listing dir contents: %s", err)
    }

    for _, fi := range files {
        if fi.IsDir() {
            continue
        }
        sfd, err := os.Open(fmt.Sprintf("%s/%s", src, fi.Name()))
        if err != nil {
            return fmt.Errorf("Failed to open src %q: %s", fi.Name(), err)
        }
        defer sfd.Close()

        data, err := ioutil.ReadAll(sfd)
        if err != nil {
            return fmt.Errorf("Failed readall of %q: %s", fi.Name(), err)
        }

        dfd, err := os.OpenFile(fmt.Sprintf("%s/%s", dst, fi.Name()), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
        if err != nil {
            return fmt.Errorf("Failed to open dst %q: %s", fi.Name(), err)
        }
        defer dfd.Close()

        fmt.Fprintf(dfd, "%s", data)
    }

    return nil
}

func init() {
    var err error

    fd, err := os.Open(fmt.Sprintf("%s/config.json", BaseDir))
    if err != nil {
        panic(err)
    }

    body, err := ioutil.ReadAll(fd)
    if err != nil {
        panic(err)
    }

    if err = json.Unmarshal(body, &config); err != nil {
        panic(err)
    } else if config.Hostname == "" || config.Username == "" || config.Password == "" || config.GitHubAccount == "" {
        panic("Not all fields are present in config.json")
    }

    // Set env variables for the terraform provider and commit binary.
    os.Setenv("PANOS_HOSTNAME", config.Hostname)
    os.Setenv("PANOS_USERNAME", config.Username)
    os.Setenv("PANOS_PASSWORD", config.Password)
}

func main() {
    var err error

    lf, err = os.OpenFile("/tmp/hook.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        panic(err)
    }
    defer lf.Close()

    log.SetOutput(lf)
    log.Printf("Starting hooksrv ...")

    http.HandleFunc("/", handleReq)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
