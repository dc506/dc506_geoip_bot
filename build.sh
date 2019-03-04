#!/bin/bash
DBS=(
https://github.com/gonet2/geoip/raw/master/GeoIP2-City.mmdb
)

for url in ${DBS[*]};do
    file=$( echo $url|sed -E 's/.+\/(.+)$/\1/g')
    [ -e "$file" ] && continue || wget $url
done

export GOPATH=$PWD
if ! [ -d ./src ]; then
    go get github.com/oschwald/geoip2-golang \
        github.com/go-telegram-bot-api/telegram-bot-api
fi

go build
