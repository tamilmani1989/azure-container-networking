
echo "Enabling IIS features"
DISM /online /enable-feature /featurename:IIS-WebServerRole
DISM /online /enable-feature /featurename:IIS-HostableWebCore
DISM /online /enable-feature /all /featurename:IIS-ASPNET
DISM /online /enable-feature /all /featurename:IIS-ASPNET45
DISM /online /enable-feature /featurename:IIS-NetFxExtensibility
DISM /online /enable-feature /featurename:IIS-NetFxExtensibility45
DISM /online /enable-feature /featurename:IIS-ISAPIExtensions
DISM /online /enable-feature /featurename:IIS-ISAPIFilter
DISM /online /enable-feature /featurename:IIS-CGI
DISM /online /enable-feature /all /featurename:IIS-ManagementScriptingTools
DISM /online /enable-feature /all /featurename:IIS-ManagementConsole
DISM /online /enable-feature /all /featurename:IIS-ManagementService
DISM /online /enable-feature /all /featurename:IIS-HttpTracing
DISM /online /enable-feature /all /featurename:IIS-RequestMonitor
DISM /online /enable-feature /all /featurename:IIS-RequestFiltering
DISM /online /enable-feature /all /featurename:IIS-WindowsAuthentication
echo "Done enabling IIS features"