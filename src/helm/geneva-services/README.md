# Geneva Services Helm Chart

## Install the chart
First, get the AME cert from keyvault that is registered with Geneva
```bash
# See what certs are available
az keyvault certificate list --vault-name acnKV

# Download the cert in base64 encoded PFX format from Keyvault using the id from above (note the /secrets instead of /certificates in id)
az keyvault secret download -f geneva-ame.pfx.b64 --id 'https://acnkv.vault.azure.net/secrets/genevaame' 
```

Convert the base64 to PFX, then to PEM files (separate cert.pem and key.pem):
```bash
openssl base64 -A -d -in geneva-ame.pfx.b64 -out geneva-ame.pfx
openssl pkcs12 -in geneva-ame.pfx -out gcscert.pem -clcerts -nokeys -passin 'pass:'
openssl pkcs12 -in geneva-ame.pfx -out gcskey.pem -nocerts -nodes -passin 'pass:'
```

Ensure the values in geneva-services/prod.values.yaml are correct. Then, set your kubectl context to the cluster you want to deploy to. To point to AKS, for example, you can use this command:
```bash
az aks get-credentials -a --resource-group test-mdnc-centraluseuap-2 --name test-mDnc-centraluseuap-2
```

Then apply the chart. (Note, set the region in the following command):

```bash
# Set your kubectl context to the cluster you want to deploy to, then:
helm upgrade genevaservices geneva-services/chart --install --values geneva-services/prod.values.yaml --set gcs_region=uscentral --set-file gcscert.pem="./gcscert.pem",gcskey.pem="./gcskey.pem"
```