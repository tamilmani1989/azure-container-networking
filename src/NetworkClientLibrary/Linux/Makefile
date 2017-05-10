TARGET	=	*.o *.so *.1.0 *.a a.out

objdir = ../../../out/linux-amd64/obj
targetdir = ../../../out/linux-amd64/targets

all: outdir library
outdir:
	mkdir -p $(objdir)
	mkdir -p $(targetdir)

library:	outdir ../NetworkClientLibrary.cpp ../Common/aquanetworkclient.h ../Common/nclcore.h ../Common/NclRepositoryManager.cpp ResponseParser.cpp Network.cpp ../../Logging/Logging.cpp
	g++ -Wno-write-strings -Wno-variadic-macros -Wall -pedantic -fPIC -I/usr/include/libxml2 -c Network.cpp ../NetworkClientLibrary.cpp ResponseParser.cpp ../Common/NclRepositoryManager.cpp ../../Logging/Logging.cpp
	g++ -Wno-write-strings -Wno-variadic-macros -shared -Wl,-soname,libAquaNetworkClient.so.1.0 -o libAquaNetworkClient.so.1.0 NetworkClientLibrary.o ResponseParser.o NclRepositoryManager.o Network.o Logging.o
	ln -sf  libAquaNetworkClient.so.1.0 libAquaNetworkClient.so
	cp libAquaNetworkClient.so.1.0 libAquaNetworkClient.so $(targetdir)
	cp *.o $(objdir)

client:	../../NetworkClient/Main.cpp ../../Logging/Logging.cpp
	g++ ../../NetworkClient/Main.cpp ../../Logging/Logging.cpp -L. -lAquaNetworkClient -lcurl -lxml2 -lstdc++

clean:	
	rm -f $(TARGET)
	rm -rf ../../../out/

