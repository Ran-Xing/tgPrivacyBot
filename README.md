## Telegram Privacy Bot

> BotFather Set Bot

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/cU96ct?referralCode=WgxCHX) [![Deploy](https://button.deta.dev/1/svg)](https://go.deta.dev/deploy?repo=https://github.com/Ran-Xing/tgPrivacyBot)

## ARG

- [ ] SEND_TO_GROUP_ID `-00000000`
- [X] TOKEN `token`
- [ ] USE_MYSQL `no/yes`
- [ ] MYSQL_CONFIG `user:name@tcp(ip:port)/tgPrivacyBot?charset=utf8mb4&parseTime=True&loc=Local`
- [ ] HTTP_PROXY, HTTPS_PROXY, NO_PROXY `http://ip:port`
- [ ] START_MESSAGE `Welcome`
- [ ] HELP_MESSAGE `help`
- [ ] HEALTH_MESSAGE `I'm OK!`
- [ ] GROUP_MESSAGE `welcome use t.me/xxxxxx`
- [X] ADMIN_ID `000000000` admin sent msg to get

### Railway

Project Settings > Tokens > Create > RAILWAY_TOKEN

### Github

Settings > Security > Secrets > Actions > New repository secret > RAILWAY_TOKEN

## Command

- /start
- /help
- /health
- /group
- /exit
- `forward text`
