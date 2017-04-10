TARGET	=	*.o *.so *.1.0 *.a

sdn:	../NetworkClientLibrary.cpp ../Common/aquanetworkclient.h ../Common/nclcore.h ../Common/NclRepositoryManager.cpp ResponseParser.cpp
	gcc -Wall -pedantic -fPIC -c ../NetworkClientLibrary.cpp ResponseParser.cpp ../Common/NclRepositoryManager.cpp
	gcc -shared -Wl,-soname,libAquaNetworkClient.so.1.0 -o libAquaNetworkClient.so.1.0 NetworkClientLibrary.o ResponseParser.o NclRepositoryManager.o
	ln -sf  libAquaNetworkClient.so.1.0 libAquaNetworkClient.so

example:	../../NetworkClient/Main.cpp
	gcc ../../NetworkClient/Main.cpp -L. -lAquaNetworkClient -lcurl -lxml2 -lstdc++

clean:	
	rm -f $(TARGET)

