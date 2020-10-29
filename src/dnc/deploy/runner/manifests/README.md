# SWIFT Runner

## Access the Kubernetes Cluster
The E2E tests run on a Kubernetes cluster. To access the cluster, you can pull down the admin kubeconfig from the runner-secrets Keyvault with az cli:

```
az keyvault secret download --name admin-kubeconfig --subscription 9bd7ff15-396a-4478-a89b-4d8ae7e302b6 --vault-name runner-secrets -f kubeconfig.yml
```

That will save the kubeconfig as `kubeconfig.yml`, which you can use by setting the `KUBECONFIG` environment variable or by using the `--kubeconfig` flag with `kubectl`:

```
kubectl --kubeconfig kubeconfig.yml get pods
```

## Secrets
The runner has secrets for datastore settings and subnet delegation tokens. These secrets are currently stored in Keyvault as well as the Kubernetes cluster as Kubernetes Secrets.

If you need to create the runner from scratch, you should pull the secret values from Keyvault and create them on the Kubernetes control plane, like so:

```
kubectl create secret generic <secret name> --from-literal=value=<value>
```

where `<secret name>` is taken from the table below, and `<value>` is the value in Keyvault.

### Secret Info
| Kubernetes Secret Name | Link to Keyvault Secret |
| ------------- | ------------- |
| storage-key | [storage-key](https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/asset/Microsoft_Azure_KeyVault/Secret/https://runner-secrets.vault.azure.net/secrets/storage-key) |
| authentication-token | [authentication-token](https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/asset/Microsoft_Azure_KeyVault/Secret/https://runner-secrets.vault.azure.net/secrets/authentication-token) |
| authentication-token-cust-vnet-with-mnat | [authentication-token-cust-vnet-with-mnat](https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/asset/Microsoft_Azure_KeyVault/Secret/https://runner-secrets.vault.azure.net/secrets/authentication-token-cust-vnet-with-mnat) |
| authentication-token-cust-vnet-with-swiftlb | [authentication-token-cust-vnet-with-swiftlb](https://ms.portal.azure.com/#@microsoft.onmicrosoft.com/asset/Microsoft_Azure_KeyVault/Secret/https://runner-secrets.vault.azure.net/secrets/authentication-token-cust-vnet-with-swiftlb) |

### Example `kubectl` Usage

```
kubectl create secret generic storage-key --from-literal=value=<storage key>
```

where `<storage key>` is the value of the secret `storage-key` in Keyvault.