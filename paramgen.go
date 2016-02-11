package main

import (
    "github.com/parnurzeal/gorequest"
    "strconv"
    "log"
    "encoding/json"
    "flag"
    "io/ioutil"
    "os/user"
    "strings"
    "text/template"
    "bytes"
    "encoding/base64"
    "os"
)

type StringParameter struct {
    Value string `json:"value"`
}

type IntParameter struct {
    Value int `json:"value"`
}

type Params struct {
    Location StringParameter `json:"location"`
    NewStorageAccountName StringParameter `json:"newStorageAccountName"`
    VmSize StringParameter `json:"vmSize"`
    NumberOfNodes IntParameter `json:"numberOfNodes"`
    AdminUserName StringParameter `json:"adminUserName"`
    SshKeyData StringParameter `json:"sshKeyData"`
    DiscoveryUrl StringParameter `json:"discoveryUrl"`
    VmNamePrefix StringParameter `json:"vmNamePrefix"`
    CustomData StringParameter `json:"customData"`
}

type AzureDeployParameters struct {
    Scheme string `json:"$schema"`
    ContentVersion string `json:"contentVersion"`
    Parameters Params `json:"parameters"`
}

func genDiscoveryUrl(size int) (string, error) {
    request := gorequest.New()
    _, body, err := request.Get("https://discovery.etcd.io/new?size=" + strconv.Itoa(size)).End()
    
    if err != nil {
        return "", err[0]
    } 
    return body, nil
}

func NewParam() *AzureDeployParameters {
    params := &AzureDeployParameters {
       Scheme: "http://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#", 
       ContentVersion: "1.0.0.0",
    }
    
    return params
}

func Init(params *AzureDeployParameters) {
    flag.StringVar(&params.Parameters.Location.Value, "location", "West US", "VM location")
    flag.StringVar(&params.Parameters.NewStorageAccountName.Value, "newStorageAccountName", "costorageaccountre", "Storage account name")
    flag.StringVar(&params.Parameters.AdminUserName.Value, "adminUserName", "core", "Admin user name")
    flag.IntVar(&params.Parameters.NumberOfNodes.Value, "numberOfNodes", 3, "Number of nodes")
    flag.StringVar(&params.Parameters.VmNamePrefix.Value, "vmNamePrefix", "core", "VM name prefix")
    flag.StringVar(&params.Parameters.VmSize.Value, "vmSize", "core", "VM Size. (default: Standard_A1)")
    flag.Parse()
    
    if params.Parameters.SshKeyData.Value == "" {
        usr, err := user.Current()
        
        if err != nil {
            log.Panic("Current use is not found", err)
        }
        
        bytes, err := ioutil.ReadFile(usr.HomeDir + "/.ssh/id_rsa.pub")
        
        if err != nil {
            log.Panic("Read default ssh key file " + usr.HomeDir + "/.ssh/id_rsa.pub failed", err)
        } else {
            params.Parameters.SshKeyData.Value = strings.Trim(string(bytes), "\n")
        }
    }
    
    if params.Parameters.DiscoveryUrl.Value == "" {
        discoveryUrl, err := genDiscoveryUrl(params.Parameters.NumberOfNodes.Value)
        
        if err != nil {
            log.Panic("DiscoveryUrl is not specified. Create new discovery url failed", err)
        }
        params.Parameters.DiscoveryUrl.Value = discoveryUrl
    }
    
    if params.Parameters.CustomData.Value == "" {
        tmp, err := template.ParseFiles("cloud-config.yaml.template")
        
        if err != nil {
            log.Panic("Error loading cloud-config template", err)
        }
        
        var cloudconf bytes.Buffer
        tmp.Execute(&cloudconf, params.Parameters)
        params.Parameters.CustomData.Value = base64.StdEncoding.EncodeToString(cloudconf.Bytes())
    }
}

func main() {
    params := NewParam()
    Init(params)
    b, err := json.MarshalIndent(params, " ", "    ")
    
    if err != nil {
        log.Panic("Error to export to json", err)
    }
    
    outputFile, err := os.Create("azuredeploy.parameters.json")
    
    if err != nil {
        log.Panic("Error to open file to write", err)
    }
    
    defer outputFile.Close()
    
    outputFile.Write(b)
}