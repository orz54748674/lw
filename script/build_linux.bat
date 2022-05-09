SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o script main.go fix.go fix2.go fixActivity.go fixGiftCode.go