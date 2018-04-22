How to compile ethvault from source and run:

install golang from golang.org (ethvault has been confirmed to work on at least 1.9.2 or 1.9.4, but later versions should also work). The site contains instructions for installing go on linux, OS X and Windows.  
The default $GOPATH set by the installer is %USERPROFILE%/go (i.e. c:/users/(your user here)/go) on Windows and /usr/local/go on linux and OS X. Open a terminal there now.  
run `go get .u github.com/vexornavy/ethvault` to download the latest version of ethvault  
run `go get -u github.com/kardianos/govendor` to download the govendor dependency management protocol  
ethvault should be in the src/github.com/vexornavy/ethvault folder relative to your current location. Go there now.  
run `go build -v -tags 'nocgo'` to build the app.  
run `./ethvault` or `./ethvault.exe` on linux/OS X or windows respectively.  
The ethvault wallet should now be running at http://localhost:8080  
