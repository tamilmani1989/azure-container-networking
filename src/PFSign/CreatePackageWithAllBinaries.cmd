set SRC=%1
set DES=%2

echo %SRC%
echo %DES%
mkdir %DES%
xcopy /s /y /h %SRC% %DES%
xcopy /s /y /h "start.bat" %DES%
xcopy /s /y /h "ServiceConfig.ini" %DES%