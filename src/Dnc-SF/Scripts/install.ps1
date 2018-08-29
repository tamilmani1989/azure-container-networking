$AppPath = ".\DNCServiceApplication"
$sdkInstallPath = (Get-ItemProperty 'HKLM:\Software\Microsoft\Service Fabric SDK').FabricSDKInstallPath
$sfSdkPsModulePath = $sdkInstallPath + "Tools\PSModule\ServiceFabricSDK"
Import-Module $sfSdkPsModulePath\ServiceFabricSDK.psm1

$StatefulServiceManifestlocation = $AppPath + "\DNCServicePkg\"
$StatefulServiceManifestlocationLinux = $StatefulServiceManifestlocation + "\ServiceManifest-Linux.xml"
$StatefulServiceManifestlocationWindows = $StatefulServiceManifestlocation + "\ServiceManifest-Windows.xml"
$StatefulServiceManifestlocationFinal= $StatefulServiceManifestlocation + "ServiceManifest.xml"
Copy-Item -Path $StatefulServiceManifestlocationWindows -Destination $StatefulServiceManifestlocationFinal -Force

$WebServiceManifestlocation = $AppPath + "\DNCServiceWebServicePkg\"
$WebServiceManifestlocationLinux = $WebServiceManifestlocation + "\ServiceManifest-Linux.xml"
$WebServiceManifestlocationWindows = $WebServiceManifestlocation + "\ServiceManifest-Windows.xml"
$WebServiceManifestlocationFinal= $WebServiceManifestlocation + "ServiceManifest.xml"
Copy-Item -Path $WebServiceManifestlocationWindows -Destination $WebServiceManifestlocationFinal -Force

Copy-ServiceFabricApplicationPackage -ApplicationPackagePath $AppPath -ApplicationPackagePathInImageStore DNCServiceApplicationType -ImageStoreConnectionString (Get-ImageStoreConnectionStringFromClusterManifest(Get-ServiceFabricClusterManifest)) -TimeoutSec 1800
Register-ServiceFabricApplicationType DNCServiceApplicationType
New-ServiceFabricApplication fabric:/DNCServiceApplication DNCServiceApplicationType 1.0.0