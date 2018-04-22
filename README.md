How to compile ethvault from source and run:

install golang from (golang.org) (ethvault has been confirmed to work on at least 1.9.2 or 1.9.4, but later versions should also work). The site contains instructions for installing go on linux, OS X and Windows.  
install git from (https://git-scm.com/downloads)
Create a workspace directory in %USERPROFILE%/go (on windows, for example, this would be C:/Users/-your user here-/go/)
run `go get -u github.com/vexornavy/ethvault` to download the latest version of ethvault (this might take a while - ignore any errors relating to missing gcc)
run `go get -u github.com/kardianos/govendor` to download the govendor dependency management tool  
ethvault should be in the src/github.com/vexornavy/ethvault folder relative to your current location. Go there now.
Make sure you have all the project's dependencies by running ´govendor sync´
run `go build -v -tags 'nocgo'` to build the app.  
run `./ethvault` or `./ethvault.exe` on linux/OS X or windows respectively.  
The ethvault wallet should now be running at http://localhost:8080  
