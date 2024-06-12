# discord-bot-cryptoprices

a simple discord bot that displays crypto status from cryptoprices.cc

![image](https://user-images.githubusercontent.com/7338312/172267762-4a725451-ac86-4f81-aa3a-6ddd88e7967c.png)

![GitHub all releases](https://img.shields.io/github/downloads/rssnyder/discord-bot/total?style=flat-square)

```text
Usage of ./discord-bot-cryptoprices:
  -activityMsg string
        bot activity
  -metrics string
        address for prometheus metric serving (default ":8080")
  -nicknameHeader string
        bot nickname
  -refresh string
        seconds between refresh (default "120")
  -setNickname string
        to update nickname (default "true")
  -status string
        0: playing, 1: listening, 2: watching (default "2")
  -symbol string
        crypto to watch
  -token string
        discord bot token
```

make sure the bot has "change nickname" permissions in the server if using that feature

## docker compose

```yaml
---
version: "3"
services:
  discord-bot:
    image: ghcr.io/rssnyder/discord-bot
    environment:
      TOKEN: XXX..XXX
      NICKNAME: some nickname
      ACTIVITY: some activity
      STATUS: 0
      REFRESH: 5
```

## helm

```
helm repo add discord-bot-cryptoprices https://rssnyder.github.io/discord-bot-cryptoprices

helm repo update discord-bot-cryptoprices

helm upgrade -i discord-bot-cryptoprices --namespace discord-bot-cryptoprices --create-namespace \
  discord-bot-cryptoprices/discord-bot-cryptoprices \
  --set token=xxxx-xxx-xxx-xxxx \
  --set symbol=ETH
```

## command line

```shell
curl -L https://github.com/rssnyder/discord-bot-cryptoprices/releases/download/v<version>/discord-bot-cryptoprices_<version>_<os>_<arch>.tar.gz -o discord-bot-cryptoprices.tar.gz
tar zxf discord-bot-cryptoprices.tar.gz

./discord-bot -token "XXX..XXX" -nickname "some nickname" -activity "some activity" -status "0" -refresh "5"
```
