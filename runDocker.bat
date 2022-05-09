SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o luckyWin main.go
docker stop 098a2da13d63
docker cp ./luckyWin 098a2da13d63:/home/goapp
docker cp ./bin/server.json 098a2da13d63:/home/goapp/bin/conf/
docker cp ./bin/lottery.json 098a2da13d63:/home/goapp/bin
docker cp ./bin/lotteryPlay.json 098a2da13d63:/home/goapp/bin
docker start 098a2da13d63