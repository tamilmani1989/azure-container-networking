Start Cns-SF wrapper on multiple node cluster:
1. Connect-ServiceFabricCluster

2. Copy-ServiceFabricApplicationPackage -ApplicationPackagePath $app -ApplicationPackagePathInImageStore CnsWrapperServiceApplicationType

3. Register-ServiceFabricApplicationType CnsWrapperServiceApplicationType

4. New-ServiceFabricApplication fabric:/CnsWrapperService CnsWrapperServiceApplicationType 1.0.0


Remove
1. Connect-ServiceFabricCluster

2. Remove-ServiceFabricApplication -ApplicationName fabric:/CnsWrapperService

3. Unregister-ServiceFabricApplicationType -ApplicationTypeName CnsWrapperServiceApplicationType -ApplicationTypeVersion 1.0.0


How to consume/extend SF-CNS-Wrapper in another branch/solution:
TODO