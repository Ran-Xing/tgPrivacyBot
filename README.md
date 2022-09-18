## Telegram Privacy Bot

> BotFather Set Bot
> 每天 3-11 不服务

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/cU96ct?referralCode=WgxCHX)

## ARG

- [ ] SEND_TO_GROUP_ID `-00000000`
- [X] TOKEN `token`
- [ ] USE_MYSQL `no/yes`
- [ ] MYSQL_CONFIG `user:name@tcp(ip:port)/tgPrivacyBot?charset=utf8mb4&parseTime=True&loc=Local`
- [ ] HTTP_PROXY, HTTPS_PROXY, NO_PROXY `http://ip:port`
- [ ] CRONTAB `0 22 * * *`
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
- `forward text`
