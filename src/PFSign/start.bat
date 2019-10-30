set ConfigFileFolder=%datadir%\\Azure\\HostNetAgent
if exist %ConfigFileFolder% ( 
    echo Config Folder Exists
) ELSE (
    echo Creating Config Directory
    mkdir %ConfigFileFolder%
)

set ConfigFilePath=%ConfigFileFolder%\\PluginSpecificTemplate.xml

:: following code checks if the pluginspecific template has already been populated, else it runs the parser.exe to generate the settings file for HostNetAgent

IF EXIST %ConfigFilePath% (
    ECHO Configuration file exists 
) ELSE (   
    ECHO Running IniParser.exe as below
    ECHO IniParser.exe PluginSpecificFileName OutFileName CloudGroupMappingHeaderFile DcmtFlattenedFile CloudName SharedIni GlobalValuesIni
    .\\INIParserBinaries\\INIParser\\IniParser.exe -pluginSpecificTemplate .\\PluginSpecificTemplate.xml -pluginSpecificOutfile %ConfigFilePath% -cloudGroupMappingHeaderFile %datadir%\\HostNetAgentDataDeployment\\HostNetAgentDCMT\\CloudGroups.ini.flattened.ini -dcmtSettingsIniFile %datadir%\\HostNetAgentDataDeployment\\HostNetAgentDCMT\\dcmtSettingsHostNetAgent.ini.flattened.ini -inputCloudName %environment% -serviceName HostNetAgent -sharedIni %datadir%\\AutopilotData\\Shared.ini.flattened.ini
    IF %errorlevel% NEQ 0 (
    ECHO Found issue with IniParser.exe %errorlevel%, exiting
    exit /b
    )
    IF NOT EXIST %ConfigFilePath% (
    ECHO Parser failed to generate the required file, with error %errorlevel%
    exit /b
    )
)

HostNetAgent.exe /configFile %ConfigFilePath% /pf

