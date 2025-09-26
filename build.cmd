@REM https://habr.com/ru/post/249449/

@SET GOOS=windows
@SET GOARCH=amd64
go build -ldflags "-s -w" -o bin/led_neima_amd64.exe

@SET GOOS=windows
@SET GOARCH=amd32
go build -ldflags "-s -w" -o bin/led_neima_amd32.exe

