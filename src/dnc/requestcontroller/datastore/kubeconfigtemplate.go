// Copyright 2020 Microsoft. All rights reserved.

package datastore

type TemplateInfo struct {
	CAData                string
	Endpoint              string
	ServerAppID           string
	ClientAppID           string
	ClientCertificatePath string
	ServerTenantID        string
	KubeloginBinaryPath   string
	AzureEnvironment      string
}

const kubeconfigTemplateString = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .Endpoint }}
  name: swift-cluster
contexts:
- context:
    cluster: swift-cluster
    user: requestcontroller
  name: swift-context
current-context: swift-context
kind: Config
preferences: {}
users:
- name: requestcontroller
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - --environment
      - {{ .AzureEnvironment }}
      - --server-id
      - {{ .ServerAppID }}
      - --client-id
      - {{ .ClientAppID }}
      - --client-certificate
      - {{ .ClientCertificatePath }}
      - --tenant-id
      - {{ .ServerTenantID }}
      - -v
      - "7"
      - --login
      - spn
      command: {{ .KubeloginBinaryPath }}`
