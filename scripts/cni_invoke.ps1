Param(
	[parameter(Mandatory=$true)]
	[string] $containerName,
	
	[parameter(Mandatory=$true)]
	[string] $namespace,
	
	[parameter(Mandatory=$true)]
	[string] $contid,
	
	[parameter (Mandatory=$true)]
	[string] $command,

	[parameter (Mandatory=$false)]
	[string] $dns

)

$env:CNI_CONTAINERID=$contid
$env:CNI_COMMAND=$command

$env:CNI_NETNS='none'
$env:CNI_PATH='C:\k\azurecni\bin'
$env:PATH="$env:CNI_PATH;"+$env:PATH
$k8sargs='IgnoreUnknown=1;K8S_POD_NAMESPACE={0};K8S_POD_NAME={1};K8S_POD_INFRA_CONTAINER_ID={2}' -f $namespace, $containerName, $contid
$env:CNI_ARGS=$k8sargs
$env:CNI_IFNAME='eth0'


$content = Get-Content -Raw -Path C:\k\azurecni\netconf\10-azure.conflist
$jobj = ConvertFrom-Json $content
$plugin=$jobj.plugins[0]
$plugin|add-member -Name "name" -Value $jobj.name -MemberType Noteproperty
$plugin|add-member -Name "cniVersion" -Value $jobj.cniVersion -MemberType Noteproperty
$arrayDataType = get-TypeData  System.Array
Remove-TypeData  System.Array
if ( $dns -ne "") {
	$serverarray = convertfrom-json $dns 
	$serverobj=@{servers=$serverarray}
	$dnsobj=@{dns=$serverobj}
	$plugin|add-member -Name "runtimeConfig" -Value $dnsobj -MemberType Noteproperty

}

$jsonconfig=ConvertTo-Json $plugin -Depth 6
echo $jsonconfig
$res=(echo $jsonconfig | azure-vnet)
echo $res
Update-TypeData -TypeData $arrayDataType 