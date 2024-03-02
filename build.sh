yarn build
go generate
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" CXX="x86_64-w64-mingw32-g++" go build -ldflags "-H windowsgui" -tags=json1 -o xbvr.exe pkg/tray/main.go
