CXX = g++
INCLUDES= -I./
CXXFLAGS = -Wall -g $(INCLUDES)
LINKFLAGS= -lcppunit -L ../../NetworkClientLibrary/Linux/ -lAquaNetworkClient -lxml2 -lcurl

NetworkClientTest: NetworkClientTest.cpp 
	$(CXX) $(CXXFLAGS) -o $@ $< $(LINKFLAGS) $(LINKFLAGSLOG4) $(LIBLOG)

# Default compile

.cpp.o:
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	rm -rf *.o
	rm -rf NetworkClientTest
